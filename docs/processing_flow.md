# Mileage Core - 処理フロー詳細

## 1. 全体リクエスト処理フロー

```mermaid
sequenceDiagram
    participant C as Client
    participant MW as Middleware Stack
    participant H as Handler
    participant UC as Usecase
    participant D as Domain
    participant DB as MySQL
    participant RD as Redis

    C->>MW: HTTP Request
    Note over MW: ① RateLimit チェック
    Note over MW: ② RequestID 生成/取得
    Note over MW: ③ Logger Start
    Note over MW: ④ Metrics 開始
    Note over MW: ⑤ JWT Auth 検証
    MW->>H: 認証済みリクエスト
    H->>UC: バリデーション済み入力
    UC->>D: ドメインロジック実行
    UC->>DB: DB操作
    UC->>RD: キャッシュ操作
    UC-->>H: 結果
    H-->>MW: レスポンス
    Note over MW: Metrics 記録
    Note over MW: Logger End
    MW-->>C: HTTP Response + X-Request-Id
```

---

## 2. ユーザー登録フロー（POST /v1/auth/signup）

```mermaid
sequenceDiagram
    participant C as Client
    participant H as AuthHandler
    participant UC as AuthUsecase
    participant UR as UserRepository
    participant MR as MileageRepository

    C->>H: POST /v1/auth/signup<br/>{email, password}
    H->>H: バリデーション<br/>(email必須, password >= 8文字)
    H->>UC: Signup(email, password)

    UC->>UR: FindByEmail(email)
    alt メール重複
        UR-->>UC: existing user
        UC-->>H: error: email already registered
        H-->>C: 409 EMAIL_CONFLICT
    end

    UC->>UC: bcrypt.GenerateFromPassword()
    UC->>UC: ULID生成
    UC->>UR: Create(user)
    UC->>MR: CreateAccount(userID)<br/>初期残高=0, version=0
    UC->>UC: JWT生成(HS256, sub=userID)
    UC-->>H: {userID, token}
    H-->>C: 201 {user_id, token}
```

---

## 3. マイル付与フロー（POST /v1/mileage/grant）

```mermaid
sequenceDiagram
    participant C as Client
    participant H as MileageHandler
    participant UC as GrantUsecase
    participant D as MileageAccount
    participant DB as MySQL
    participant RD as Redis

    C->>H: POST /v1/mileage/grant<br/>Idempotency-Key: xxx<br/>{amount: 100, reason: "daily_login"}
    H->>H: バリデーション<br/>(amount > 0, Idempotency-Key必須)
    H->>UC: Execute(input)

    Note over UC: ── 冪等性チェック ──
    UC->>DB: FindTransactionByIdempotencyKey<br/>(userID, key, "grant")

    alt 同一キー取引が存在
        alt 同一amount
            DB-->>UC: existing transaction
            UC->>DB: GetAccount(userID)
            UC-->>H: 既存結果を返す（冪等）
            H-->>C: 200 {transaction_id, balance, version}
        else 異なるamount
            UC-->>H: ErrIdempotencyConflict
            H-->>C: 409 IDEMPOTENCY_CONFLICT
        end
    end

    Note over UC: ── 楽観ロック付きTx（最大3回リトライ）──
    loop attempt = 1..3
        UC->>DB: BEGIN TX
        UC->>DB: GetAccount(userID)<br/>SELECT balance, version
        DB-->>UC: {balance: 500, version: 5}

        UC->>D: account.Grant(100)
        Note over D: balance += 100 → 600<br/>version++ → 6<br/>不変条件: amount > 0 ✓

        UC->>DB: InsertTransaction<br/>(ULID, userID, 100, "grant", key)
        Note over DB: UNIQUE(user_id, key, type)<br/>重複なら ErrAlreadyProcessed

        UC->>DB: UpdateBalance<br/>SET balance=600, version=version+1<br/>WHERE user_id=? AND version=5
        alt version一致
            DB-->>UC: RowsAffected = 1
            UC->>DB: COMMIT
        else version不一致（誰かが先に更新）
            DB-->>UC: RowsAffected = 0
            UC->>DB: ROLLBACK
            Note over UC: ErrOptimisticLock → リトライ
        end
    end

    Note over UC: ── キャッシュ無効化 ──
    UC->>RD: DEL mileage:balance:{userID}

    Note over UC: ── Metrics記録 ──
    UC-->>H: {transactionID, balance: 600, version: 6}
    H-->>C: 200 {transaction_id, balance, version}
```

---

## 4. マイル消費フロー（POST /v1/mileage/redeem）

```mermaid
sequenceDiagram
    participant C as Client
    participant H as MileageHandler
    participant UC as RedeemUsecase
    participant D as MileageAccount
    participant DB as MySQL
    participant RD as Redis

    C->>H: POST /v1/mileage/redeem<br/>Idempotency-Key: yyy<br/>{amount: 200, reason: "coupon"}
    H->>H: バリデーション

    Note over UC: 冪等性チェック（Grantと同じ）

    loop 楽観ロック付きTx（最大3回）
        UC->>DB: SELECT balance, version
        DB-->>UC: {balance: 600, version: 6}

        UC->>D: account.Redeem(200)
        alt 残高不足 (balance < amount)
            D-->>UC: ErrInsufficientBalance
            UC-->>H: error
            H-->>C: 409 INSUFFICIENT_BALANCE
        end
        Note over D: balance -= 200 → 400<br/>version++ → 7

        UC->>DB: INSERT mileage_transactions
        UC->>DB: UPDATE SET balance=400 WHERE version=6
        alt 成功
            UC->>DB: COMMIT
        else 競合
            UC->>DB: ROLLBACK → リトライ
        end
    end

    UC->>RD: DEL mileage:balance:{userID}
    UC-->>H: {transactionID, balance: 400, version: 7}
    H-->>C: 200
```

---

## 5. 残高照会フロー（GET /v1/mileage/balance）

```mermaid
sequenceDiagram
    participant C as Client
    participant H as MileageHandler
    participant UC as BalanceUsecase
    participant RD as Redis
    participant DB as MySQL

    C->>H: GET /v1/mileage/balance
    H->>UC: Execute(userID)

    UC->>RD: GET mileage:balance:{userID}
    alt キャッシュヒット
        RD-->>UC: {balance: 400, version: 7}
        UC-->>H: 即座に返却
        H-->>C: 200 {user_id, balance: 400, version: 7}
    else キャッシュミス
        RD-->>UC: nil
        UC->>DB: GetAccount(userID)
        DB-->>UC: {balance: 400, version: 7}
        UC->>RD: SET mileage:balance:{userID}<br/>TTL: 5分
        UC-->>H: DB結果を返却
        H-->>C: 200 {user_id, balance: 400, version: 7}
    end
```

---

## 6. 取引履歴フロー（GET /v1/mileage/transactions）

```mermaid
sequenceDiagram
    participant C as Client
    participant H as MileageHandler
    participant UC as TransactionsUsecase
    participant DB as MySQL

    C->>H: GET /v1/mileage/transactions?limit=20&cursor=
    H->>UC: Execute(userID, cursor, limit=20)

    UC->>DB: SELECT ... WHERE user_id=?<br/>ORDER BY created_at DESC<br/>LIMIT 21 (limit+1)
    DB-->>UC: 21件取得

    Note over UC: 先頭20件 → items<br/>21件目のID → next_cursor

    UC-->>H: {items: [...20件], next_cursor: "01J..."}
    H-->>C: 200
```

---

## 7. エラーハンドリングフロー

```mermaid
flowchart TD
    E[エラー発生] --> T{エラー種別}

    T -->|ErrInvalidAmount| B400[400 BAD_REQUEST]
    T -->|ErrInsufficientBalance| C409A[409 INSUFFICIENT_BALANCE]
    T -->|ErrOptimisticLock| RETRY{リトライ回数}
    T -->|ErrIdempotencyConflict| C409B[409 IDEMPOTENCY_CONFLICT]
    T -->|ErrAlreadyProcessed| IDEM[冪等レスポンス返却<br/>元の結果をそのまま]
    T -->|JWT無効| U401[401 UNAUTHORIZED]
    T -->|レート制限| R429[429 RATE_LIMIT_EXCEEDED]
    T -->|その他| S500[500 INTERNAL_ERROR]

    RETRY -->|< 3回| LOOP[ループ先頭へ]
    RETRY -->|>= 3回| C409C[409 OPTIMISTIC_LOCK_CONFLICT]

    style B400 fill:#f59e0b
    style C409A fill:#ef4444
    style C409B fill:#ef4444
    style C409C fill:#ef4444
    style U401 fill:#6366f1
    style R429 fill:#8b5cf6
    style S500 fill:#dc2626
    style IDEM fill:#10b981
```

---

## 8. データ整合性の保証まとめ

```mermaid
flowchart LR
    subgraph "二重実行防止"
        IK["Idempotency-Key<br/>UNIQUE(user_id, key, type)"]
        IK --> IC{同一キー存在?}
        IC -->|Yes, 同一内容| SR[同じ結果を返す]
        IC -->|Yes, 別内容| ER[409 エラー]
        IC -->|No| PROC[通常処理へ]
    end

    subgraph "競合制御"
        OL["楽観ロック<br/>UPDATE WHERE version=?"]
        OL --> VER{version一致?}
        VER -->|Yes| OK[COMMIT成功]
        VER -->|No| RT[ROLLBACK → リトライ]
    end

    subgraph "キャッシュ一貫性"
        WR[Write完了] --> INV[Redis DEL]
        RD[Read] --> HIT{Cache Hit?}
        HIT -->|Yes| FAST[即座に返却]
        HIT -->|No| DBRD[DB読み取り → Cache SET]
    end

    PROC --> OL
```

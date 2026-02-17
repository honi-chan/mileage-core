# Mileage Core - Bruno API テストガイド

## セットアップ

1. [Bruno](https://www.usebruno.com/) をインストール
2. Bruno を起動 → **Open Collection** → `mileage-core/bruno` を選択

## コレクション構成

```
bruno/
├── collection.bru                  # 共通変数（baseUrl, token）
├── Health Check.bru
├── auth/
│   ├── Signup.bru                  # token 自動保存
│   └── Login.bru                   # token 自動保存
├── mileage/
│   ├── Get Balance.bru
│   ├── Grant Mileage.bru
│   ├── Redeem Mileage.bru
│   └── Get Transactions.bru
└── achievement/
    └── Create Achievement.bru
```

## 共通変数

| 変数 | デフォルト値 | 説明 |
|------|------------|------|
| `baseUrl` | `http://localhost:8081` | APIサーバーURL |
| `token` | （自動設定） | Signup/Login で自動保存されるJWT |

---

## テスト実行手順

### Step 1: サーバー起動

```bash
PORT=8081 make run
```

### Step 2: Signup → token 取得

**auth/Signup** を実行。レスポンス例：

```json
{
  "user_id": "01JMC...",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

> `token` は `collection.bru` の変数に自動保存される。
> 以降のリクエストで `Authorization: Bearer {{token}}` として自動適用。

### Step 3: マイル付与

**mileage/Grant Mileage** を実行。

| フィールド | 値 | 備考 |
|-----------|-----|------|
| amount | 100 | 付与マイル数 |
| reason | daily_login | 理由 |
| Idempotency-Key | grant-001 | **実行ごとに変更** |

### Step 4: 残高確認

**mileage/Get Balance** を実行。

### Step 5: マイル消費

**mileage/Redeem Mileage** を実行。

| フィールド | 値 | 備考 |
|-----------|-----|------|
| amount | 50 | 消費マイル数 |
| reason | coupon | 理由 |
| Idempotency-Key | redeem-001 | **実行ごとに変更** |

### Step 6: 取引履歴

**mileage/Get Transactions** を実行。

---

## テストシナリオ

### 冪等性の確認

1. **Grant Mileage** を `Idempotency-Key: test-idem-001` で実行
2. 同じキーでもう一度実行 → **同じ結果が返る**（二重付与されない）
3. 同じキーで `amount` を変えて実行 → **409 IDEMPOTENCY_CONFLICT**

### 残高不足テスト

1. **Get Balance** で現在残高を確認
2. **Redeem Mileage** で残高を超える amount を指定
3. → **409 INSUFFICIENT_BALANCE**

### 認証テスト

1. `token` 変数を空にして **Get Balance** 実行
2. → **401 UNAUTHORIZED**

---

## エラーレスポンス一覧

| HTTP | エラーコード | 発生条件 |
|------|------------|----------|
| 400 | BAD_REQUEST | バリデーションエラー |
| 401 | UNAUTHORIZED | JWT無効・期限切れ |
| 409 | INSUFFICIENT_BALANCE | 残高不足 |
| 409 | IDEMPOTENCY_CONFLICT | 同キー別内容 |
| 429 | RATE_LIMIT_EXCEEDED | レート制限超過 |

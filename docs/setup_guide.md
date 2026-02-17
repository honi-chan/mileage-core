# Mileage Core - セットアップ手順

## 前提条件

- Go 1.22+
- Docker Desktop（MySQL コンテナ稼働済み）
- mysql CLI

---

## 1. 既存 MySQL コンテナの確認

```bash
docker ps --format "table {{.Names}}\t{{.Ports}}\t{{.Status}}" | grep mysql
```

```
mysql   0.0.0.0:3306->3306/tcp   Up 13 days (healthy)
```

> ポート3306が既に使用中のため、`docker-compose.yml` の MySQLは起動不可。
> 既存コンテナ（root/rootpass）を流用する。

## 2. データベース・ユーザー作成

```bash
docker exec mysql mysql -u root -prootpass -e \
  "CREATE DATABASE IF NOT EXISTS mileage CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
   CREATE USER IF NOT EXISTS 'mileage'@'%' IDENTIFIED BY 'mileage';
   GRANT ALL PRIVILEGES ON mileage.* TO 'mileage'@'%';
   FLUSH PRIVILEGES;"
```

## 3. テーブル作成（マイグレーション）

```bash
make migrate
```

または直接：

```bash
mysql -h 127.0.0.1 -P 3306 -u mileage -pmileage mileage < migrations/001_create_tables.up.sql
```

作成されるテーブル：

| テーブル | 用途 |
|----------|------|
| `users` | ユーザー（email, password_hash） |
| `mileage_accounts` | マイレージ口座（balance, version） |
| `mileage_transactions` | 取引履歴（冪等性キー付き） |
| `achievement_events` | 行動イベント |

## 4. Redis 起動

```bash
docker compose up -d redis
```

## 5. API サーバー起動

```bash
make run
```

`http://localhost:8080` で起動。

## 6. 動作確認

```bash
# ヘルスチェック
curl localhost:8080/health

# ユーザー登録
curl -s -X POST localhost:8080/v1/auth/signup \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"password123"}' | jq
```

---

## ロールバック

```bash
make migrate-down
# または
mysql -h 127.0.0.1 -P 3306 -u mileage -pmileage mileage < migrations/001_create_tables.down.sql
```

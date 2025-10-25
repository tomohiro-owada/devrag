---
title: "Claude Codeに無料のRAG導入でトークン＆時間節約"
emoji: "🤖"
type: "tech"
topics: ["claudecode", "rag", "ai", "開発効率化", "mcp"]
published: true
---

## TL;DR

Claude Code に毎回ドキュメントを読ませるのやめませんか？

**ローカルで動く無料の RAG** (DevRag) を使えば、Claude Code がベクトル検索で勝手にドキュメントを見つけてくれます。何百とあるドキュメントのファイル名や場所を、私たちが覚える必要はありません。

- **完全無料**: API 不要、ローカル完結
- **簡単**: 5 分でセットアップ完了
- **高速**: トークン消費 1/40、レスポンス 15 倍速
- リポジトリ: https://github.com/tomohiro-owada/devrag （作者が Claude Code で毎日困ってたので作りました）

## Claude Code にドキュメントを読ませる 3 つの問題

### 1. コンテキストがもったいない

Claude Code のコンテキストウィンドウには限りがあります。ドキュメントを丸ごと読ませるたびに、大量のトークンが消えていきます。

```
あなた: 「プロジェクトのAPI認証方式を確認して」
Claude: *Readツールでdocs/auth.mdを読み込む (3,000トークン消費)*
Claude: 「JWTベースの認証を使用しています」
```

この 3,000 トークン、次の質問では使えません。同じドキュメントについて聞きたいことがあっても、また最初から読み直しです。

### 2. どのファイルか探すのが大変

ドキュメントが増えてくると、Claude 自身も「どこに何が書いてあるか」がわかりません。

```
あなた: 「Redisのキャッシュ戦略について教えて」
Claude: *とりあえず読んでみる*
       - docs/architecture.md (4,000トークン)
       - docs/caching.md (2,000トークン)
       - docs/redis.md (存在しない)

実際に必要だったのは docs/caching.md の一部（200トークン分）だけ
```

ドキュメントが 10 個、20 個と増えると、当てずっぽうでファイルを読むことになります。

**特にチーム開発だと深刻です。**

- 他の人が書いたドキュメントの場所なんて知らない
- プロジェクトに参加したばかりだと、どこに何があるか全くわからない
- ドキュメントが 50 個、100 個とあったら、探すだけで一苦労

「あのAPIの仕様、どこに書いてあったっけ？」と毎回探し回ることになります。

### 3. 同じことを何度も繰り返す

プロジェクトで同じドキュメントを参照することって、結構ありますよね。

```
セッション1: "認証について" → docs/auth.md を Read (3,000トークン)
セッション2: "認証エラー対処" → docs/auth.md を Read (3,000トークン)
セッション3: "認証のテスト" → docs/auth.md を Read (3,000トークン)
```

**同じファイルを 3 回読んで 9,000 トークン消費**

毎回全文読み込むので、必要な情報が後半にあっても最初から読み直しです。

## RAG で全部解決する

RAG（Retrieval-Augmented Generation）を使えば、これらの問題が一気に解決します。

### 仕組みはシンプル

1. **最初に一回だけ**: ドキュメントをベクトル化して DB 保存
2. **質問するとき**: 関連する部分だけをベクトル検索で取得
3. **回答生成**: 必要な情報だけを使って Claude が回答

```
【従来】
質問 → ドキュメント全体を読む (3,000トークン) → 回答

【RAG】
質問 → 関連部分のみ検索 (200トークン) → 回答
```

**トークン消費を 1/15 に削減**できて、**検索精度も上がります**。

一番大きいのは、**私たちがファイル名を知らなくても Claude Code が勝手に見つけてくれる**こと。何百とあるドキュメントの中から、質問に関連する情報を自動で引っ張ってきてくれます。

## DevRag: Claude Code 専用の簡易 RAG

Claude Code で使うために、できるだけシンプルな RAG を作りました。

### 特徴

- **ワンバイナリー**: Python も外部 DB も不要
- **自動セットアップ**: モデルは初回起動時に勝手にダウンロード
- **MCP 統合**: Claude Code に`search`ツールとして追加されるだけ
- **高速**: 起動 2 秒、検索 100ms 以下
- **多言語対応**: 日本語も英語も問題なし

### セットアップは 5 分

#### 1. バイナリダウンロード

[Releases](https://github.com/tomohiro-owada/devrag/releases)から環境に合ったファイルをダウンロード：

```bash
# macOS (Apple Silicon)
wget https://github.com/tomohiro-owada/devrag/releases/latest/download/devrag-macos-apple-silicon.tar.gz
tar -xzf devrag-macos-apple-silicon.tar.gz
chmod +x devrag-macos-apple-silicon
sudo mv devrag-macos-apple-silicon /usr/local/bin/devrag
```

#### 2. Claude Code 設定

`~/.claude.json` に追加：

```json
{
  "mcpServers": {
    "devrag": {
      "type": "stdio",
      "command": "/usr/local/bin/devrag"
    }
  }
}
```

#### 3. ドキュメント置き場を作る

```bash
mkdir documents
cp your-notes.md documents/
```

これで終わり。あとは起動時に勝手にインデックス化されます。

## 実際に使ってみる

### Before: 従来の方法

```
あなた: 「このプロジェクトのDBマイグレーション方法は？」

Claude: *Readツールで順番に読む*
- README.md (5,000トークン)
- docs/database.md (4,000トークン)
- docs/setup.md (3,000トークン)

合計: 12,000トークン消費
時間: 約30秒
```

ファイル名を知らないと、関連しそうなファイルを全部読むことになります。

### After: DevRag 使用

```
あなた: 「DBマイグレーション方法は？」

Claude: *searchツールでベクトル検索*
検索結果: docs/database.md の関連部分のみ (300トークン)

「`npm run migrate` でマイグレーションを実行します。
詳細は docs/database.md:42 を参照してください」

合計: 300トークン消費
時間: 約2秒
```

## まとめ

Claude Code にドキュメントを読ませるのは：

- ❌ コンテキストがもったいない
- ❌ ファイルを探すのが大変
- ❌ 同じことを何度も繰り返す

RAG を使えば：

- ✅ トークン消費 **1/40**
- ✅ レスポンス **15 倍速**
- ✅ **ファイル名を知らなくても Claude Code が見つけてくれる**
- ✅ セットアップ **5 分**、完全無料

何百もあるドキュメントの場所を覚える必要はありません。Claude Code にベクトル検索で勝手に探してもらいましょう。

---

**リポジトリ**: https://github.com/tomohiro-owada/devrag
**ライセンス**: MIT
**フィードバック**: [Issues](https://github.com/tomohiro-owada/devrag/issues)

ぜひ試してみてください！

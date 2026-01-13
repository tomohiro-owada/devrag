# DevRag ソースコードインデックス機能 仕様書

## 目次

1. [概要](#概要)
2. [要件定義](#要件定義)
3. [システムアーキテクチャ](#システムアーキテクチャ)
4. [依存ライブラリ](#依存ライブラリ)
5. [対応言語とシンボル抽出](#対応言語とシンボル抽出)
6. [データベーススキーマ](#データベーススキーマ)
7. [API仕様](#api仕様)
8. [使用方法](#使用方法)
9. [内部実装詳細](#内部実装詳細)
10. [制限事項と注意点](#制限事項と注意点)

---

## 概要

DevRagのソースコードインデックス機能は、Tree-sitterを用いたAST（抽象構文木）解析により、ソースコードを意味的な単位（関数、クラス、メソッド等）でチャンク化し、ベクトル検索可能にする機能です。

### 背景と目的

従来のマークダウンインデックスでは、段落やヘッダー単位でのチャンク化を行っていましたが、ソースコードに対しては以下の課題がありました：

- 行数ベースの分割では関数の途中で切れてしまう
- コードの意味的な境界が無視される
- 関数名やクラス名などのメタデータが失われる

本機能では、Tree-sitterによるAST解析を用いることで、**関数・クラス・メソッド単位での意味的なチャンク化**を実現し、より精度の高いコード検索を可能にします。

### 主な機能

| 機能 | 説明 |
|------|------|
| AST解析 | Tree-sitterによる構文解析 |
| シンボル抽出 | 関数、メソッド、クラス、構造体、インターフェースの自動認識 |
| メタデータ保存 | シンボル名、型、行番号、シグネチャの保存 |
| ベクトル検索 | 埋め込みベクトルによる意味的検索 |

---

## 要件定義

### 機能要件

#### FR-1: 対応プログラミング言語

| ID | 要件 | 優先度 |
|----|------|--------|
| FR-1.1 | Go言語（.go）のインデックス化に対応する | 必須 |
| FR-1.2 | Python（.py）のインデックス化に対応する | 必須 |
| FR-1.3 | TypeScript（.ts, .tsx）のインデックス化に対応する | 必須 |
| FR-1.4 | JavaScript（.js, .jsx）のインデックス化に対応する | 必須 |

#### FR-2: シンボル抽出

| ID | 要件 | 優先度 |
|----|------|--------|
| FR-2.1 | 関数定義を抽出する | 必須 |
| FR-2.2 | メソッド定義を抽出する | 必須 |
| FR-2.3 | クラス定義を抽出する | 必須 |
| FR-2.4 | 構造体定義を抽出する（Go） | 必須 |
| FR-2.5 | インターフェース定義を抽出する | 必須 |
| FR-2.6 | アロー関数を抽出する（TypeScript/JavaScript） | 必須 |

#### FR-3: メタデータ

| ID | 要件 | 優先度 |
|----|------|--------|
| FR-3.1 | シンボル名を保存する | 必須 |
| FR-3.2 | シンボルタイプを保存する | 必須 |
| FR-3.3 | 開始行・終了行を保存する | 必須 |
| FR-3.4 | 関数シグネチャを保存する | 必須 |
| FR-3.5 | 親シンボル（メソッドの場合のクラス名等）を保存する | 推奨 |

#### FR-4: MCPツール

| ID | 要件 | 優先度 |
|----|------|--------|
| FR-4.1 | `index_code`ツールでソースコードをインデックス化できる | 必須 |
| FR-4.2 | 既存の`search`ツールでコード検索ができる | 必須 |
| FR-4.3 | `delete_document`ツールでコードを削除できる | 必須 |
| FR-4.4 | `index_code`でディレクトリ単位のインデックスができる | 必須 |
| FR-4.5 | `index_code`で複数ファイルを一括インデックスできる | 必須 |
| FR-4.6 | 差分同期（変更ファイルのみ再インデックス）ができる | 必須 |
| FR-4.7 | 強制再インデックスオプションがある | 推奨 |

### 非機能要件

| ID | 要件 | 目標値 |
|----|------|--------|
| NFR-1 | インデックス処理速度 | 100チャンク/秒以上 |
| NFR-2 | 検索レスポンス時間 | 500ms以下 |
| NFR-3 | メモリ使用量 | 500MB以下 |
| NFR-4 | 同一シンボルの重複排除 | 100% |

### 制約事項

| ID | 制約 | 理由 |
|----|------|------|
| C-1 | CGOが必要 | Tree-sitterはCライブラリを使用 |
| C-2 | 対応言語はTree-sitterの言語バインディングに依存 | 言語追加にはTree-sitter文法が必要 |
| C-3 | 巨大な関数は分割されない | AST単位での整合性を維持 |

---

## システムアーキテクチャ

### コンポーネント構成

```
┌─────────────────────────────────────────────────────────────────┐
│                         MCP Server                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    index_code Tool                       │   │
│  └───────────────────────────┬─────────────────────────────┘   │
│                              │                                  │
│  ┌───────────────────────────▼─────────────────────────────┐   │
│  │                      Indexer                             │   │
│  │  ┌─────────────────┐  ┌─────────────────────────────┐   │   │
│  │  │  CodeParser     │  │  Embedder                   │   │   │
│  │  │  (Tree-sitter)  │  │  (ONNX Runtime)             │   │   │
│  │  └────────┬────────┘  └─────────────┬───────────────┘   │   │
│  │           │                         │                    │   │
│  │  ┌────────▼────────────────────────▼───────────────┐    │   │
│  │  │              LanguageConfig                      │    │   │
│  │  │  (Go, Python, TypeScript, JavaScript)            │    │   │
│  │  └─────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│  ┌───────────────────────────▼─────────────────────────────┐   │
│  │                     VectorDB                             │   │
│  │  ┌──────────┐ ┌──────────┐ ┌────────────────────────┐   │   │
│  │  │documents │ │ chunks   │ │ code_metadata          │   │   │
│  │  └──────────┘ └──────────┘ └────────────────────────┘   │   │
│  │  ┌─────────────────────────────────────────────────┐    │   │
│  │  │              vec_chunks (sqlite-vec)             │    │   │
│  │  └─────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 処理フロー

```
┌──────────────────┐
│ index_code呼出し │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ 言語検出         │
│ (拡張子から判定) │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ ファイル読込     │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Tree-sitter解析  │
│ (AST生成)        │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ シンボル抽出     │
│ (クエリ実行)     │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ 埋め込み生成     │
│ (バッチ処理)     │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ データベース保存 │
│ (トランザクション)│
└──────────────────┘
```

### ファイル構成

```
internal/
├── indexer/
│   ├── chunk.go        # CodeChunk型定義
│   ├── languages.go    # 言語設定とTree-sitterクエリ
│   ├── code.go         # CodeParser実装
│   ├── code_test.go    # ユニットテスト
│   └── indexer.go      # IndexCodeFile実装
├── vectordb/
│   ├── schema.go       # code_metadataテーブル定義
│   ├── db.go           # InsertCodeDocument実装
│   └── search.go       # 検索機能
└── mcp/
    ├── server.go       # ツール登録
    └── tools.go        # index_codeツール実装
```

---

## 依存ライブラリ

### Tree-sitter関連

| パッケージ | バージョン | 用途 |
|-----------|-----------|------|
| `github.com/smacker/go-tree-sitter` | v0.0.0-20240827094217-dd81d9e9be82 | Tree-sitter Go バインディング |

#### 言語パーサー（インポート）

```go
import (
    sitter "github.com/smacker/go-tree-sitter"
    "github.com/smacker/go-tree-sitter/golang"
    "github.com/smacker/go-tree-sitter/javascript"
    "github.com/smacker/go-tree-sitter/python"
    "github.com/smacker/go-tree-sitter/typescript/typescript"
)
```

### その他の依存ライブラリ

| パッケージ | バージョン | 用途 |
|-----------|-----------|------|
| `github.com/asg017/sqlite-vec-go-bindings` | v0.1.6 | ベクトル検索 |
| `github.com/yalue/onnxruntime_go` | v1.21.0 | 埋め込み生成 |
| `github.com/mark3labs/mcp-go` | v0.42.0 | MCPプロトコル |
| `github.com/mattn/go-sqlite3` | v1.14.32 | SQLiteドライバ |

### インストール方法

```bash
# Tree-sitter依存の追加
go get github.com/smacker/go-tree-sitter
go get github.com/smacker/go-tree-sitter/golang
go get github.com/smacker/go-tree-sitter/python
go get github.com/smacker/go-tree-sitter/typescript/typescript
go get github.com/smacker/go-tree-sitter/javascript

# ビルド（CGO必須）
CGO_ENABLED=1 go build -o devrag ./cmd/main.go
```

### ビルド要件

| 要件 | 説明 |
|------|------|
| Go | 1.23以上 |
| CGO | 有効（sqlite-vec、Tree-sitter用） |
| Cコンパイラ | gcc/clang |

---

## 対応言語とシンボル抽出

### Go (.go)

#### 抽出対象シンボル

| シンボルタイプ | Tree-sitterクエリ | 例 |
|---------------|-------------------|-----|
| function | `(function_declaration name: (identifier) @name) @function` | `func Hello()` |
| method | `(method_declaration name: (field_identifier) @name) @method` | `func (u *User) Greet()` |
| struct | `(type_declaration (type_spec name: (type_identifier) @name type: (struct_type))) @struct` | `type User struct{}` |
| interface | `(type_declaration (type_spec name: (type_identifier) @name type: (interface_type))) @interface` | `type Reader interface{}` |

#### シグネチャ抽出例

```go
// 入力
func Hello(name string) string {
    return "Hello, " + name
}

// 抽出されるシグネチャ
"func Hello(name string) string"
```

### Python (.py)

#### 抽出対象シンボル

| シンボルタイプ | Tree-sitterクエリ | 例 |
|---------------|-------------------|-----|
| function | `(function_definition name: (identifier) @name) @function` | `def greet(name):` |
| class | `(class_definition name: (identifier) @name) @class` | `class User:` |

#### シグネチャ抽出例

```python
# 入力
def greet(name):
    return f"Hello, {name}!"

# 抽出されるシグネチャ
"def greet(name)"
```

### TypeScript (.ts, .tsx)

#### 抽出対象シンボル

| シンボルタイプ | Tree-sitterクエリ | 例 |
|---------------|-------------------|-----|
| function | `(function_declaration name: (identifier) @name) @function` | `function hello()` |
| function (arrow) | `(lexical_declaration (variable_declarator name: (identifier) @name value: (arrow_function))) @function` | `const hello = () => {}` |
| class | `(class_declaration name: (type_identifier) @name) @class` | `class User {}` |
| interface | `(interface_declaration name: (type_identifier) @name) @interface` | `interface User {}` |
| method | `(method_definition name: (property_identifier) @name) @method` | クラス内メソッド |

#### シグネチャ抽出例

```typescript
// 通常関数
function formatUser(user: User): string {
    return user.name;
}
// シグネチャ: "function formatUser(user: User): string"

// アロー関数
const fetchUser = async (id: number): Promise<User> => {
    return { id, name: "Test" };
};
// シグネチャ: "const fetchUser = async (id: number): Promise<User> =>"
```

### JavaScript (.js, .jsx)

#### 抽出対象シンボル

| シンボルタイプ | Tree-sitterクエリ | 例 |
|---------------|-------------------|-----|
| function | `(function_declaration name: (identifier) @name) @function` | `function hello()` |
| function (arrow) | `(lexical_declaration (variable_declarator name: (identifier) @name value: (arrow_function))) @function` | `const hello = () => {}` |
| class | `(class_declaration name: (identifier) @name) @class` | `class User {}` |
| method | `(method_definition name: (property_identifier) @name) @method` | クラス内メソッド |

---

## データベーススキーマ

### code_metadataテーブル

```sql
CREATE TABLE IF NOT EXISTS code_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chunk_id INTEGER NOT NULL UNIQUE,
    symbol_name TEXT,
    symbol_type TEXT NOT NULL,
    language TEXT NOT NULL,
    start_line INTEGER,
    end_line INTEGER,
    parent_symbol TEXT,
    signature TEXT,
    FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE CASCADE
);

-- インデックス
CREATE INDEX IF NOT EXISTS idx_code_symbol ON code_metadata(symbol_name);
CREATE INDEX IF NOT EXISTS idx_code_language ON code_metadata(language);
CREATE INDEX IF NOT EXISTS idx_code_type ON code_metadata(symbol_type);
```

### フィールド説明

| フィールド | 型 | 説明 |
|-----------|-----|------|
| id | INTEGER | 主キー |
| chunk_id | INTEGER | chunksテーブルへの外部キー |
| symbol_name | TEXT | シンボル名（関数名、クラス名等） |
| symbol_type | TEXT | function, method, class, struct, interface |
| language | TEXT | go, python, typescript, javascript |
| start_line | INTEGER | 開始行（1始まり） |
| end_line | INTEGER | 終了行（1始まり） |
| parent_symbol | TEXT | 親シンボル（メソッドの場合のクラス名） |
| signature | TEXT | 関数シグネチャ |

### ER図

```
┌─────────────┐       ┌─────────────┐       ┌─────────────────┐
│  documents  │       │   chunks    │       │  code_metadata  │
├─────────────┤       ├─────────────┤       ├─────────────────┤
│ id (PK)     │◄──┐   │ id (PK)     │◄──────│ id (PK)         │
│ filename    │   └───│ document_id │       │ chunk_id (FK)   │
│ indexed_at  │       │ position    │       │ symbol_name     │
│ modified_at │       │ content     │       │ symbol_type     │
└─────────────┘       └──────┬──────┘       │ language        │
                             │              │ start_line      │
                      ┌──────▼──────┐       │ end_line        │
                      │  vec_chunks │       │ parent_symbol   │
                      ├─────────────┤       │ signature       │
                      │ embedding   │       └─────────────────┘
                      │ FLOAT[384]  │
                      └─────────────┘
```

---

## API仕様

### MCPツール: index_code

`index_code`ツールは3つのモードをサポートし、差分更新（変更ファイルのみ再インデックス）に対応しています。

#### 動作モード

| モード | パラメータ | 説明 |
|--------|-----------|------|
| 単一ファイル | `filepath` | 1つのファイルをインデックス |
| ディレクトリ | `directory` | 配下の全コードファイルを差分同期 |
| 複数ファイル | `filepaths` | 指定した複数ファイルをインデックス |

#### パラメータ

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| filepath | string | いずれか1つ | 単一コードファイルのパス（.go, .py, .ts, .tsx, .js, .jsx） |
| directory | string | いずれか1つ | ディレクトリパス。配下の全コードファイルを差分同期 |
| filepaths | string | いずれか1つ | 複数ファイルパス（カンマ区切り）。例: `main.go,utils.go` |
| force | boolean | いいえ | 強制再インデックス。trueの場合は変更有無に関わらず再インデックス（デフォルト: false） |

#### 対応拡張子

- `.go` - Go
- `.py` - Python
- `.ts`, `.tsx` - TypeScript
- `.js`, `.jsx` - JavaScript

#### リクエスト例

**モード1: 単一ファイル**

```json
{
  "method": "tools/call",
  "params": {
    "name": "index_code",
    "arguments": {
      "filepath": "/path/to/main.go"
    }
  }
}
```

**モード2: ディレクトリ（差分同期）**

```json
{
  "method": "tools/call",
  "params": {
    "name": "index_code",
    "arguments": {
      "directory": "/path/to/src"
    }
  }
}
```

**モード3: 複数ファイル**

```json
{
  "method": "tools/call",
  "params": {
    "name": "index_code",
    "arguments": {
      "filepaths": "main.go,handler.go,utils.go"
    }
  }
}
```

**強制再インデックス**

```json
{
  "method": "tools/call",
  "params": {
    "name": "index_code",
    "arguments": {
      "filepath": "/path/to/main.go",
      "force": true
    }
  }
}
```

#### レスポンス（単一ファイル成功時）

```json
{
  "success": true,
  "message": "Code file indexed successfully",
  "file": "/path/to/main.go"
}
```

#### レスポンス（ディレクトリ成功時）

```json
{
  "success": true,
  "message": "Directory indexed: 15 files processed",
  "indexed": 15,
  "skipped": 3,
  "failed": 0,
  "added": 10,
  "updated": 5,
  "directory": "/path/to/src"
}
```

#### レスポンス（複数ファイル成功時）

```json
{
  "success": true,
  "message": "Files indexed: 3 processed",
  "indexed": 3,
  "skipped": 0,
  "failed": 0,
  "added": 2,
  "updated": 1
}
```

#### レスポンス（エラー時）

```json
{
  "error": "unsupported file type. Supported: [.go .py .ts .tsx .js .jsx]"
}
```

### 差分同期の仕組み

ディレクトリモードおよび複数ファイルモードでは、以下の差分同期が自動的に行われます：

1. **新規ファイル**: データベースに存在しないファイルをインデックス
2. **更新ファイル**: ファイル更新日時がデータベースの記録より新しい場合に再インデックス
3. **未変更ファイル**: スキップ（処理時間を短縮）

`force=true`を指定すると、変更有無に関わらず全ファイルを再インデックスします。

### 検索結果のコードメタデータ

`search`ツールの結果には、コードチャンクの場合、以下のメタデータが含まれます：

```json
{
  "results": [
    {
      "filename": "/path/to/source.go",
      "content": "func GetUser(id int) (*User, error) { ... }",
      "similarity": 0.85,
      "metadata": {
        "symbol_name": "GetUser",
        "symbol_type": "function",
        "language": "go",
        "start_line": 45,
        "end_line": 52,
        "signature": "func GetUser(id int) (*User, error)"
      }
    }
  ]
}
```

---

## 使用方法

### 基本的な使用例

#### 1. 単一ファイルのインデックス

```
User: このGoファイルをインデックスして
      /path/to/main.go

Claude: index_codeツールを使用してインデックスします。

[index_code filepath="/path/to/main.go" 実行]

インデックスが完了しました。以下のシンボルが抽出されました：
- 関数: 5個
- メソッド: 3個
- 構造体: 2個
```

#### 2. ディレクトリ全体のインデックス（差分同期）

```
User: srcディレクトリ配下のコードを全部インデックスして

Claude: index_codeツールでディレクトリを差分同期します。

[index_code directory="/path/to/src" 実行]

ディレクトリのインデックスが完了しました：
- インデックス済み: 15ファイル
- スキップ（未変更）: 3ファイル
- 新規追加: 10ファイル
- 更新: 5ファイル
```

#### 3. 複数ファイルのインデックス

```
User: main.goとhandler.goとutils.goをインデックスして

Claude: index_codeツールで複数ファイルをインデックスします。

[index_code filepaths="main.go,handler.go,utils.go" 実行]

3ファイルのインデックスが完了しました。
```

#### 4. 強制再インデックス

```
User: main.goを強制的に再インデックスして

Claude: forceオプションを使用して再インデックスします。

[index_code filepath="main.go" force=true 実行]

強制再インデックスが完了しました。
```

#### 5. コードの検索

```
User: ユーザー認証に関する関数を検索して

Claude: searchツールで検索します。

[search実行]

以下の関数が見つかりました：
1. authenticateUser (auth.go:23-45)
   シグネチャ: func authenticateUser(username, password string) (*User, error)

2. validateToken (token.go:12-30)
   シグネチャ: func validateToken(token string) bool
```

### ディレクトリ内の複数ファイルのインデックス

現在のバージョンでは、ファイル単位でのインデックスのみ対応しています。複数ファイルをインデックスする場合は、各ファイルに対して`index_code`を実行してください。

```bash
# シェルスクリプトでの一括インデックス例
find . -name "*.go" -exec devrag index_code {} \;
```

### Claude Codeでの推奨ワークフロー

1. **プロジェクト初期設定時**
   - 主要なソースファイルをインデックス
   - 設定ファイル（config.json）でドキュメントディレクトリを設定

2. **開発中**
   - 変更したファイルを再インデックス
   - `reindex_document`ツールで更新

3. **コードレビュー時**
   - 関連する関数を検索
   - 類似実装を検索

---

## 内部実装詳細

### CodeChunk構造体

```go
type SymbolType string

const (
    SymbolTypeFunction  SymbolType = "function"
    SymbolTypeMethod    SymbolType = "method"
    SymbolTypeClass     SymbolType = "class"
    SymbolTypeStruct    SymbolType = "struct"
    SymbolTypeInterface SymbolType = "interface"
)

type CodeChunk struct {
    Content      string      // コード本文
    Position     int         // ドキュメント内位置（0始まり）
    SymbolName   string      // シンボル名
    SymbolType   SymbolType  // シンボルタイプ
    Language     string      // プログラミング言語
    StartLine    int         // 開始行（1始まり）
    EndLine      int         // 終了行（1始まり）
    ParentSymbol string      // 親シンボル（メソッドの場合）
    Signature    string      // 関数シグネチャ
}
```

### 埋め込みテキストフォーマット

コードチャンクは以下のフォーマットで埋め込みベクトルに変換されます：

```
"{language} {symbol_name}: {content}"
```

例：
```
"go GetUser: func (s *UserService) GetUser(id int) (*User, bool) {
    user, exists := s.users[id]
    return user, exists
}"
```

このフォーマットにより、言語とシンボル名が埋め込みに含まれ、検索精度が向上します。

### 重複排除ロジック

同一シンボルの重複を防ぐため、バイト位置をキーとしたマップで管理：

```go
seen := make(map[string]bool)
key := fmt.Sprintf("%d-%d", startByte, endByte)
if seen[key] {
    return nil  // 重複をスキップ
}
seen[key] = true
```

### シグネチャ抽出ロジック

言語ごとに異なる抽出ロジックを適用：

| 言語 | 抽出方法 |
|------|---------|
| Go | 最初の`{`より前を抽出 |
| Python | 最初の`:`より前を抽出 |
| TypeScript/JavaScript | `{`または`=>`より前を抽出 |

---

## 制限事項と注意点

### 現在の制限事項

| 制限 | 説明 | 回避策 |
|------|------|--------|
| 言語サポート | Go, Python, TypeScript, JavaScriptのみ | 他言語は将来追加予定 |
| ディレクトリ一括インデックス | 未対応 | ファイル単位で実行 |
| 巨大関数の分割 | 未対応 | 関数全体が1チャンクとして保存 |
| ネストした関数 | 部分的対応 | 言語により動作が異なる |
| コメント・ドキュメント | 関数本体に含まれる | 別途抽出は未対応 |

### パフォーマンス考慮事項

| 項目 | 推奨値 | 備考 |
|------|--------|------|
| ファイルサイズ | 1MB以下 | 大きいファイルは処理時間増加 |
| 同時インデックス | 1ファイル | トランザクション競合回避 |
| チャンク数上限 | なし | メモリ使用量に注意 |

### エラーハンドリング

| エラー | 原因 | 対処 |
|--------|------|------|
| "unsupported language" | 非対応の拡張子 | 対応言語のファイルを指定 |
| "invalid path" | パストラバーサル検出 | 設定済みディレクトリ内のパスを指定 |
| "failed to parse" | 構文エラー | ファイルの構文を確認 |
| "[WARN] No symbols extracted" | シンボルなし | 空ファイルまたは対応外の構文 |

### セキュリティ考慮事項

- **パストラバーサル対策**: `validatePath`関数で設定済みディレクトリ外へのアクセスを防止
- **入力検証**: ファイルパスとパラメータの検証を実施
- **CGO安全性**: Tree-sitterのCバインディングはメモリ安全

---

## 付録

### A. Tree-sitterクエリ構文

```
(node_type) @capture_name
(parent (child) @name) @parent
(declaration name: (identifier) @name) @capture
```

- `node_type`: ASTノードタイプ
- `@capture_name`: キャプチャ名（コードから参照）
- `name:`: フィールドセレクタ

### B. テストデータ

テストデータは `test_data/code/` に配置：

- `sample.go` - Go言語サンプル
- `sample.py` - Pythonサンプル
- `sample.ts` - TypeScriptサンプル

### C. 参考資料

- [Tree-sitter公式ドキュメント](https://tree-sitter.github.io/tree-sitter/)
- [go-tree-sitter GitHub](https://github.com/smacker/go-tree-sitter)
- [DevRag CLAUDE.md](../CLAUDE.md)
- [GitHub Issue #9](https://github.com/tomohiro-owada/devrag/issues/9)

---

## 変更履歴

| バージョン | 日付 | 変更内容 |
|-----------|------|----------|
| 1.1.0 | 2025-12-22 | ディレクトリ・複数ファイル対応、差分同期機能追加 |
| 1.0.0 | 2025-12-22 | 初版作成 |

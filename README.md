# mackerel-plugin-linux-process-status

mackerel metric plugin for CPU/FDs/Memory usage of each linux process.

各LinuxプロセスのCPU・ファイルディスクリプタ・メモリ使用量を監視し、mackerelにメトリクスを送信するためのプラグインです。

## 目次

- [特徴](#特徴)
- [インストール](#インストール)
- [使い方](#使い方)
- [取得できるメトリクス](#取得できるメトリクス)
- [動作環境](#動作環境)
- [制限事項](#制限事項)

## 特徴

- **プロセス単位での詳細なメトリクス取得**: PIDを指定することで、個々のプロセスの資源使用状況を詳細に監視できます
- **軽量な設計**: `procfs`を直接参照する仕組みにより、最小限のオーバーヘッドでメトリクスを取得できます
- **ファイルディスクリプタの監視**: 開いているFD数とシステム制限値の割合を追跡できます
- **メモリ使用量の可視化**: プロセスが使用するメモリ量とシステム全体のメモリに対する割合を監視できます

## インストール

### mkr コマンドを使用する場合（推奨）

[mkr](https://github.com/mackerelio/mackerel-agent) コマンドがインストールされている場合、次のコマンドでプラグインをインストールできます：

```bash
mkr plugin install monitoring-forge/mackerel-plugin-linux-process-status
```

### リリースからダウンロードする場合

[GitHub Releasesページ](https://github.com/monitoring-forge/mackerel-plugin-linux-process-status/releases) から、お使いの環境に適したバイナリをダウンロードして使用してください。


## 使い方

### コマンドラインオプション

```
Usage:
  mackerel-plugin-linux-process-status [OPTIONS]

Application options:
  -p, --pid=        PID                監視対象のプロセスのPID
      --key-prefix=                     メトリクスキーのプレフィックス
  -v, --version                         バージョン情報を表示

Help options:
  -h, --help                            ヘルプメッセージを表示
```

### 基本的な使い方

監視対象のプロセスのPIDを指定して実行します。`--key-prefix` オプションでメトリクスキーのプレフィックスを指定できます（省略不可）。

```bash
./mackerel-plugin-linux-process-status --key-prefix postgres -p 54321
```

### 使用例

PostgreSQLプロセス（PID: 54321）を監視する場合：

```bash
./mackerel-plugin-linux-process-status --key-prefix postgres -p 54321
```

出力例：

```
process-status.fds_postgres.count   6       1606457504
process-status.fds_postgres.max     65535   1606457504
process-status.fds_usage_postgres.percentage        0.009155        1606457504
process-status.cpu_postgres.percentage      0.000000        1606457504
process-status.mem_postgres.used    2924544 1606457504
process-status.mem_postgres.max     469286912       1606457504
process-status.mem_usage_postgres.percentage        0.623189        1606457504
```

> **注意**: 初回実行時はCPU使用率のメトリクスは出力されません。2回目以降の実行で、前回の状態との差分からCPU使用率が計算されます。

### pgrep / pidof と組み合わせて PID を動的に指定する

PID が起動のたびに変わるサービスを監視する場合、`pgrep` や `pidof` などのコマンドと組み合わせて PID を動的に取得できます。

#### `pgrep` を使う例

プロセス名に一致する最初の PID を取得します：

```bash
./mackerel-plugin-linux-process-status --key-prefix postgres -p "$(pgrep -f postgres)"
```

> **注意**: `pgrep` が複数の PID を返す場合、このプラグインは単一の PID のみを受け付けるため、`-n`（最新のプロセス）や `-o`（最も古いプロセス）などのオプションで1つに絞るか、 `head` を使ってください。
>
> ```bash
> ./mackerel-plugin-linux-process-status --key-prefix postgres -p "$(pgrep -n postgres)"
> ./mackerel-plugin-linux-process-status --key-prefix postgres -p "$(pgrep postgres | head -n 1)"
> ```

#### `pidof` を使う例

`pidof` は実行ファイル名に一致する PID を返します：

```bash
./mackerel-plugin-linux-process-status --key-prefix nginx -p "$(pidof nginx)"
```

`pidof` も複数の PID を返すことがあるため、単一に絞る場合は以下のようにします：

```bash
./mackerel-plugin-linux-process-status --key-prefix nginx -p "$(pidof nginx | awk '{print $1}')"
```

#### mackerel-agent 設定例

mackerel-agent では、シェルのコマンド置換を利用して PID を動的に渡せます。

```yaml
[plugin.metrics.postgres]
command = "./mackerel-plugin-linux-process-status --key-prefix postgres -p $(pgrep -n postgres)"

[plugin.metrics.nginx]
command = "./mackerel-plugin-linux-process-status --key-prefix nginx -p $(pidof nginx | awk '{print $1}')"
```

## 取得できるメトリクス

プラグインは以下のメトリクスを出力します。各メトリクスにはタイムスタンプ（UNIX時間）が最後に付与されます。

### ファイルディスクリプタ関連

| メトリクス名 | 説明 | 単位 |
|---|---|---|
| `process-status.fds_{prefix}.count` | プロセスが開いているファイルディスクリプタの数 | 整数 |
| `process-status.fds_{prefix}.max` | システムが許可する最大ファイルディスクリプタ数 | 整数 |
| `process-status.fds_usage_{prefix}.percentage` | 使用可能なファイルディスクリプタの使用率（%） | 割合（%） |

**使用例**:
- `count` が `6` で、`max` が `65535` の場合、使用率 `percentage` は約 `0.009%` になります
- 使用率が上昇傾向にある場合は、ファイルディスクリプドリークの兆候である可能性があります

### CPU使用量関連

| メトリクス名 | 説明 | 単位 |
|---|---|---|
| `process-status.cpu_{prefix}.percentage` | プロセスのCPU使用率（ユーザー空間＋システム空間） | 割合（%） |

**計算方法**:
- プラグインは2回以上の実行間で、プロセスのCPU時間とシステム全体のCPU時間の差分から使用率を計算します
- この仕組みにより、正確なCPU使用率の割合を取得できます

### メモリ使用量関連

| メトリクス名 | 説明 | 単位 |
|---|---|---|
| `process-status.mem_{prefix}.used` | プロセスが使用しているメモリ量（RSS: Resident Set Size） | バイト |
| `process-status.mem_{prefix}.max` | システム全体の総メモリ量（MemTotal） | バイト |
| `process-status.mem_usage_{prefix}.percentage` | プロセスのメモリ使用量占总メモリの割合 | 割合（%） |

**注意**:
- `used` はプロセスのRSS（実物理メモリ使用量）を表します
- `max` はシステム全体の物理メモリ量（`/proc/meminfo` の `MemTotal`）を使用します
- cgroup制限には対応していません。コンテナ環境では注意が必要です

## 動作環境

- **OS**: Linuxのみ（Windows/macOSでは動作しません）
- **カーネル**: procfsが使用可能な環境（通常のリナックスディストリビューションで対応）
- **Go**: 1.25.0以上（ソースビルドする場合）

## 制限事項

- **I/O待機場所含むCPU時間の監視には対応していません**
- **CPU steal time（仮想化環境でのハイパーバイザによるCPU時間の占有）には対応していません**
- Linuxのみの対応です。WindowsやmacOSでは動作しません
- cgroup（コンテナ）の制限には対応していません。`MemTotal` を最大値として使用するため、コンテナ環境では正確なメモリ使用率が表示されない場合があります
- 初回実行時はCPU使用率のメトリクスは出力されません。2回目以降の実行で差分計算が行われます
- プラグインのワークディレクトリにステータスファイル（JSON）が保存されます。複数プロセスで同じ`--key-prefix`を指定した場合、ファイルが上書きされる可能性があります

### セキュリティに関する注意

- プラグインは `/proc/{pid}` を直接読み取るため、実行ユーザーに適切な権限が必要です
- 通常、`root` または対象プロセスと同じユーザーで実行する必要があります

## ライセンス

[MIT License](LICENSE)


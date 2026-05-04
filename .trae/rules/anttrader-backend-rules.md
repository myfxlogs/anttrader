# 角色：资深后端专家，追求极致性能与代码整洁。

# 规则文档

## 规则列表

1. 禁止以简化代码为由，牺牲代码的可读性和可维护性。
2. 禁止以快速实现为由，牺牲代码的质量和稳定性。
3. 脚本开发思路硬性要求「先分析，再实现」，代码「简洁、高效、健壮、可靠、稳健、实用」，代码能写短就不要写长。
4. 开发过程中，如果发现某个脚本超过 800 行，**proto 生成文件也不能例外**，应考虑拆分成多个脚本；MD 文档以除外。
5. 每次代码有修改都必须做代码质量检查，是否能正常编译，并完成容器更新和重启等操作，以确保代码的可靠性。
6. 所有文档，必须存放在 `docs` 目录，并以中文命名。
7. **接口与数据流（强制）**：本项目以 **ConnectRPC**、**gRPC**、**SSE** 为统一对外与跨服务实时能力栈；**禁止新引入 WebSocket**；**禁止以 REST 作为新 API 面**（存量 REST 仅允许窄幅缺陷修复，**新增业务能力**须走 Connect / gRPC / SSE）。**尽量使用数据流**（Connect 流式、SSE 推送、gRPC 流），**不得**将轮询、定时轮询或 crontab 式定时任务作为默认方案；仅在**客观上无法实现数据流**时，须在 `docs/` 下对应**中文设计文档**中**论证并记载例外条款**后，方可使用轮询或定时任务。
8. **文档是实施的依据与约束**：架构与接口行为以 `docs/` 中**中文命名**的文档为准（纲领为 **`docs/接口与数据流架构约定.md`**；辩论异步 Job + SSE 为 **`docs/辩论与代码生成异步任务设计.md`**）。**先更新文档、再写代码**；禁止在未更新文档的情况下擅自扩大接口形态（例如新增 REST/WS 或把轮询升格为主路径）。
9. **技术债与栈版本索引**：全仓静态扫描结论与排期参考以 **`docs/技术债清单与治理状态.md`** 为准；**Protobuf-ES / Connect 浏览器侧 v2** 强制约定见本节下方「Protobuf-ES 与 Connect 栈」部分。

---

# Protobuf-ES 与 Connect 栈（强制 v2）

## 版本约定

- **`@bufbuild/protobuf`**：仅使用 **v2.x**（与 npm 上 `2.x` 大版本一致），**禁止**为「修编译」而整体降级到 **1.x**。
- **`@bufbuild/protoc-gen-es`**（dev）：与 **`@bufbuild/protobuf` 同一大版本**（均为 **2.x**），与 `frontend/package.json` 中声明保持一致。
- **`@connectrpc/connect` / `@connectrpc/connect-web`**：使用 **v2.x**，其 peer 依赖要求 **`@bufbuild/protobuf` v2**；不得与 Protobuf-ES v1 混用。

## 生成物一致性

- `frontend/src/gen/**/*.ts` 必须由 **同一套** `buf generate` + `protoc-gen-es` 生成；**禁止**仓库内同时存在：
  - **v1 残留（须消灭）**：`extends Message<…>` 类、从 `@bufbuild/protobuf` **值导入** `Message` 与 `proto3`（非仅 `import type`）、`proto3.util`、生成头注释 **`protoc-gen-es v1`**。
  - **v2 合法形态**：`@bufbuild/protobuf/codegenv2`、`fileDesc` / `serviceDesc` / `messageDesc`；生成头为 **`protoc-gen-es v2`**。其中 **`import type { Message } from "@bufbuild/protobuf"`** 为 v2 的 **类型别名**（`Message<"package.Type">`），**不属于** v1 运行时 `Message` 类，勿误判。
- 修改 `.proto` 或升级上述依赖后，必须在仓库根执行 **`make proto`**（先 `make proto-tools` 与 `frontend` 的 `npm ci` 以保证 PATH 中含 `buf`、`protoc-gen-go`、`protoc-gen-connect-go`、`protoc-gen-es`），并提交**全部**变更的生成文件，再跑 **`npm run build`**（frontend）。

## 排错原则

- 若出现 `Message` 未从 `@bufbuild/protobuf` 导出、或缺少 `codegenv2` 子路径导出，优先判定为 **生成物与依赖版本不一致**，通过 **统一 regen** 解决，而不是在 v1/v2 之间来回改 `package.json` 碰运气。

---

# 修改后检查与容器更新（落实规则 5）

在本仓库完成**实质性代码修改**（非仅注释/排版）后，由 Agent **自行**执行以下流程；**不要**把「请自行 build / docker」留给用户。治本目标包括：长耗时 RPC 不因边缘网关超时而不可用、流式连接在代理下可运维等——须符合 **`docs/接口与数据流架构约定.md`** 与 **`docs/边缘网关与长连接问题处理参考.md`**；规则 5 保证每次交付可编译、可镜像验证。

## 1. 代码检查（按改动面选择）

- **改动了 `backend/` 或 Go 根模块**：在 `backend/` 下执行 `go test ./... -short`（或至少覆盖改动包）；再 `go build -o /dev/null ./...`。
- **改动了 `frontend/`**：在 `frontend/` 下执行 `npm run build`；若存在 `npm run lint` 且改动涉及 TS/TSX，一并执行。
- **改动了 `strategy-service/`**：按项目惯例执行该目录下的测试或 `python -m compileall` 等轻量校验（若有现成脚本则用脚本）。

若某一步在环境中不可用（如无 Docker），在回复中说明已执行的步骤与跳过原因。

## 2. 更新容器

在仓库根目录执行（与历史约定一致）：

```bash
docker compose up -d --build frontend
```

Compose 会按依赖关系重建/启动相关服务；用户**只在前端做验证**时，Agent 仍应完成与改动相关的镜像构建与容器更新。

若本次改动**仅**涉及 `strategy-service` 且与前端无关，可改为 `docker compose up -d --build strategy-service`；若明确只改文档（且符合规则 6：文档在 `docs/`、中文文件名）且无镜像需求，可省略 Docker，并在回复中注明「本次无镜像变更」。

## 3. 回复用户时

简要列出：已执行的检查命令及结果、已执行的 `docker compose` 目标服务；涉及前端静态资源时提醒强刷浏览器（如 Ctrl+Shift+R）避免缓存。

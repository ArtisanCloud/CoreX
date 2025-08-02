# CoreX 项目规范文档（路径 / 命名 / 责任说明）

## 一、总体原则

1. **后端统一 snake\_case**

   * 所有目录、包名、Go 源文件、小写下划线分词（例如 `agent_tools`、`event_bus`、`flow_executor.go`）。
   * 不使用驼峰、合并词、混用。
   * 代码内部的类型/结构体/接口仍用 Go 约定的驼峰（例如 `ExecutionPlan`, `Agent`, `EventBus`）。

2. **前端资源用 kebab-case**

   * 例如 `workflow-builder`、`plugin-ui`，用短横线区分前端静态组件与后端模块。

3. **配置 / YAML / 字段** 用 snake\_case；**环境变量** 用大写下划线（如 `CORE_X_LICENSE_KEY`）；**前端组件** 用 kebab-case。

---

## 二、命名细则速查

| 项目        | 规范示例                                        |
| --------- | ------------------------------------------- |
| Go 包 / 目录 | `agent_tools`, `event_bus`, `dynamic_form`      |
| Go 文件     | `flow_executor.go`, `feature_flag.go`       |
| Go 测试文件   | `flow_executor_test.go`                     |
| 类型/接口/结构体 | `EventBus`, `LowCodeFlow`                   |
| 配置字段      | `flow_definition`, `trigger_event`          |
| 环境变量      | `CORE_X_LICENSE_KEY`, `AGENT_TOOLS_ENABLED` |
| 前端资源      | `workflow-builder`, `plugin-host`           |

---

## 三、Lint 与 CI 校验建议

引入自动命名检查，CI 中执行避免规范回退，可参考以下脚本（集成到 PR 流程）：

```bash
#!/usr/bin/env bash
set -e
bad=0

echo "检查目录/文件命名是否符合 snake_case..."

# 目录
find . -type d | while read -r dir; do
  base=$(basename "$dir")
  [[ "$base" =~ [A-Z] ]] && echo "目录含大写: $dir" && bad=1
  [[ "$base" =~ [^a-z0-9_] ]] && echo "目录含非法字符: $dir" && bad=1
done

# Go 文件
find . -name '*.go' | while read -r file; do
  name=$(basename "$file")
  [[ "$name" =~ [A-Z] ]] && echo "Go 文件含大写: $file" && bad=1
  [[ ! "$name" =~ ^[a-z0-9_]+\.go$ ]] && echo "Go 文件名不符合 snake_case: $file" && bad=1
done

if [[ $bad -ne 0 ]]; then
  echo "命名规范检查未通过" >&2
  exit 1
fi
echo "命名规范检查通过"
```

---

## 四、完整目录树（原样保留）与每项详细作用

```bash
/CoreX/
  go.mod                          # go module 定义（版本、依赖、对外暴露 module path）
  /cmd/
    /demo/                        # 示例启动器（参考实现，用于本地调试、集成测试）
      main.go                     # 引导 bootstrap：初始化 plugin, agent_tools, event_bus，挂载 internal/middleware，启动 HTTP/gRPC/WebSocket/SSE 服务，构建 tenant/trace/log 上下文
  /pkg/                           # 暴露给下游的能力库（可被 PowerX / MediaX 直接 import 复用）
    /auth/                        # 认证与鉴权
      middleware.go              # HTTP/gRPC 中间件：token 解析/校验（JWT、API Key）、权限判断、注入 tenant_id/user_id 等上下文
      session.go                 # session 管理（生成/刷新 token）、可扩展 SSO（如 OIDC、内部 session store）、续期与失效控制
    /license/                     # license 与 feature gating
      checker.go                # license 解析、有效性验证（Basic/Pro/灰度）、多租户限额/限制、生效期、版本策略
      feature_flag.go           # 细粒度功能开关控制：基于 license 决定 extension, tool, feature 是否可用（支持灰度/分段开启）
    /plugin/                      # 插件系统核心（extension point 架构）
      loader.go                 # 插件加载器：发现、初始化、生命周期管理（singleton 模式）、依赖注入与验证
      registry.go               # 插件注册中心：管理 extension point 与已注册实现、查询/启用/禁用插件
      extension_point.go        # 插件接口定义：如 on_tag_applied、on_flow_completed、on_plan_updated 等 hook 规范
      base.go                   # 插件元信息结构、标准 lifecycle 接口（init/validate/run/teardown）、错误处理约定
    /agent/                     # 抽象 contract 层 + driver 实现（智能体能力编排执行）
      /contract/               # 核心接口定义（ExecutionPlan、Agent、Planner、Executor）
        planner.go          # Planner 接口 + 相关 plan 构造输入 helper（意图到计划）
        executor.go         # Executor 接口 + 执行入口定义
        agent.go            # Agent 接口（组合 Planner + Executor + 高阶方法）
        execution_plan.go   # ExecutionPlan、PlanStep 等计划结构
        retry_policy.go     # RetryPolicy 定义
        result.go           # ToolResult / StepResult 等执行结果封装
        errors.go           # 共享标准 error 变量
      /factory/
        agent.go        # 通用注册 & 构造逻辑
      /drivers/                # 各 Agent 实现驱动（可扩展多种智能体风格）
        /eino/                # 默认 Eino 实现（意图→plan→flow/node 执行→反馈）
          plan.go              # ExecutionPlan 构造逻辑，自然语言意图解析、fallback 策略、优先级、验证、上下文融合
          execution.go         # driver 层执行器：短路(short-circuit)、flow 调度、并行、降级、事件发布（构建在 core flow/node 基础原语之上）
          flow.go              # 复合 flow 定义与编排（条件分支、嵌套 flow、子任务组合）
          node.go              # driver 封装的 node：包装 core node、附加 intent/context 处理、适配输入输出与 side-effect
          feedback.go          # 多轮反馈机制：订阅 core 事件、分析执行结果、调整或再生成 plan
          prompt_template.md  # few-shot prompt 模板：execution plan 生成样板、约束示例、上下文格式说明
          agent.go       # 实现 contract.Agent 与 Planner 的封装（整合 plan→execute→feedback 流程，提供统一调用接口）
        # future other agents: /rule_based/, /keyword_agent/, etc.  # 预留可插拔智能体驱动
    /event_bus/                  # 事件总线（内部解耦与联动）
      bus.go                   # 发布/订阅核心、事件结构定义、基础过滤、优先级调度、幂等校验、重试逻辑
      subscriber.go           # 订阅封装：条件订阅、失败降级、重试策略、幂等性保障（包装下游 handler）
    /dynamic_form/                  # 低代码动态 flow/form 执行引擎
      schema.go               # flow/form 定义 schema（trigger、condition、steps、vars、outputs、error handling 结构）
      form_executor.go        # 表单解析/校验/映射、输入转换、默认值注入、与 flow step 绑定执行
    /comm/                      # 实时通信层（状态 & 事件推送）
      /websocket/              # WebSocket 实现
        hub.go                # 连接管理、主题订阅、广播、分类路由、trace id 关联
        handler.go            # WebSocket 协议升级入口、session 绑定、权限核验
        auth_middleware.go    # WebSocket 特化鉴权：复用 auth 逻辑注入上下文
        message.go            # 统一封包格式：事件类型、trace metadata、payload 结构
      /sse/                    # Server-Sent Events（轻量化单向订阅）
        manager.go            # 客户端订阅管理、心跳/恢复、状态 diff 计算
        handler.go            # HTTP 接口，建立持久连接、推送增量事件
    /utils/                    # 通用辅助工具
      logger.go               # 结构化日志封装（trace id、level、context fields、输出格式）
      validator.go           # schema 校验、组合规则、错误汇总
      mask.go                # 脱敏策略（基于角色/视图：admin vs miniapp）、字段级处理
      context_key.go         # context key 常量定义（避免字符串冲突与 typo）
  /internal/                     # 不暴露的实现细节（支撑层，业务不可直接 import，走 pkg / api 访问）
    /storage/                   # 持久化接口抽象（接口定义，可插 GORM / mock / pluggable store，实现替换与测试隔离）
    /http/                      # 共享 HTTP 封装与路由组合逻辑
      router.go                # 通用路由组生成器：挂载 middleware、版本控制、子路由聚合
      middleware.go            # （如果存在的 legacy 聚合点）可封装 error handling / request logging / feature injection，建议逐步迁移到细化 internal/middleware 模块
  /api/                         # CoreX 可运行时暴露的接口（对外调用入口，组合底层能力）
    /http/
      router.go                # 高阶路由组合（agent/tool/orchestrator/health），把 internal middleware 挂载，处理路由分层与版本
      /agent/
        tool_controller.go     # REST Proxy 执行 tool：POST /api/v1/agent/tools/{name}，统一输入输出规范、权限校验、审计
        orchestrator_controller.go # 启动复杂 flow：从自然语言 prompt 映射、生成执行计划并派发执行
      health.go                # 子系统探活接口：license 状态、plugin 加载状况、event_bus 健康、依赖链路检测
    /grpc/
      /v1/
        agent_service.proto    # gRPC contract 定义：ExecuteTool、StartFlow、GetFlowStatus、流状态回传等 RPC 接口
        agent_service.pb.go     # 生成的 Go stub（通过 protoc 生成的类型、安全封装）
      /interceptors/
        auth_interceptor.go    # gRPC 认证/上下文注入（复用 pkg/auth 逻辑、session 验证）
        license_interceptor.go # feature gating 拦截（请求前预判可用能力）
        error_mapper.go        # 内部 error -> gRPC status 映射统一化（兼容客户端规范）
  /docs/                        # 配套文档（示例 + 规范）——人/AI 共用契约
    style_guide.md             # 命名/结构/编码风格（本规范），包含常见反模式、prompt 约定、示例生成指令
    flow_schema.md            # dynamic_form flow/form 结构定义详情 + 变量模板（JSON/YAML 版 schema）
    tool_contracts.md         # agent_tools 输入输出、权限、错误格式约定（调用方与实现方的契约）
    plugin_development.md     # 插件如何实现 & 注册 extension point（生命周期、hook 规范、扩展点文档）
    agent_prompt_templates.md # 智能体 prompt → flow/tool 映射模板（few-shot 实例、上下文注入格式、fallback 策略描述）
    license_and_gating.md     # license 格式 / 特性开关 / Pro gating 规则（灰度、限额、版本策略）
    deployment.md            # 嵌入式 vs 远程服务部署指南（依赖图、启动顺序、健康检查、回滚机制）
```

---

## 五、关键子系统说明（完整）

1. **cmd/demo**

   * 负责系统启动：聚合各子系统（plugin/agent/event\_bus/middleware/API server），注入全局 context（tenant、trace、用户身份、日志），并暴露 HTTP/gRPC/WebSocket/SSE 接口。
   * 示例与调试入口，供开发环境与集成测试复用。

2. **pkg/auth**

   * 提供认证、鉴权、session lifecycle、角色权限、上下文注入能力。
   * 输出供 middleware 与 API 层使用的身份信息（tenant\_id、user\_id、scopes）。

3. **pkg/license**

   * 解读 license 内容（Basic/Pro/灰度/限额），输出 feature\_flag 状态。
   * 驱动 extension gating（控制 plugin/tool/agent 某些能力是否开放）。

4. **pkg/plugin**

   * 提供扩展点定义与加载机制。
   * 插件能响应核心事件（如 flow 结束、tag 变动）、修改执行路径、触发外部同步（例如 CRM 同步）。

5. **pkg/agent**

   * contract 层定义智能体执行语义与 plan 协议。
   * driver 层（如 Eino）从自然语言意图生成 ExecutionPlan，调度 flow/node 执行，并通过 feedback 形成闭环优化。
   * 支持多种 agent 实现可插拔，保持统一契约。

6. **pkg/event\_bus**

   * 事件路由与消费解耦机制。
   * 提供可靠投递（重试/幂等）、条件订阅、异步执行、插件/flow/agent 绑定。

7. **pkg/low\_code**

   * 用数据描述旅程（flow definition），并对步骤/表单进行执行控制、分支、变量穿透与错误恢复。

8. **pkg/comm**

   * 实时通信：WebSocket 处理双向、人机交互状态流；SSE 适用于轻量单向 dashboard 订阅。
   * 标准化消息格式、鉴权、连接与恢复策略。

9. **pkg/utils**

   * 统一日志、input/schema 验证、脱敏、上下文 key 管理、错误包装，避免各模块重复实现。

10. **internal/storage**

    * 持久化接口抽象，可替换实现（测试 mock、实际 GORM、分布式 store），封装事务/连接管理惯例。

11. **internal/http**

    * 组合路由、版本化、子服务挂载点，是将 middleware 与 controller 经过组织后形成的实际 HTTP 接口底层。

12. **internal/middleware**（推荐代替 monolithic middleware.go）

    * 认证、license gating、日志、错误规范处理等跨切面能力。由 bootstrap 统一注册挂载到 API 路由。

13. **api/http**

    * 面向外部的 REST 接口层，暴露 agent/tool/orchestrator/health 等高阶能力。
    * 负责组成业务级路由并调用底层 pkg/internal 能力，统一请求/响应格式及权限规则。

14. **api/grpc**

    * 高性能服务间/外部调用入口。
    * 定义标准 RPC contract（ExecuteTool、StartFlow 等），用 interceptors 注入身份/feature gating，并把内部错误映射成 gRPC status。

15. **docs/**

    * 形成“人+机”共识的规范仓库：风格、契约、示例、prompt、插件、license 策略、部署，支撑开发、AI 代码生成与运维一致性。

---

## 六、典型跨模块协作流程（举例：客户标签流转）

1. 客户画像通过 agent 读取（走 `api/http` → middleware 注入身份/feature gating → `pkg/agent` 调用 `get_customer_profile` tool）。
2. Agent 决策后调用 `apply_tag` tool，触发 `on_tag_applied` 插件（`pkg/plugin` 处理同步逻辑）。
3. 基于标签结果，Agent 启动一个复购 flow（`start_flow`），由 `pkg/dynamic_form` 执行器调度步骤。
4. flow 中的状态与插件反馈通过 `pkg/event_bus` 发散，前端通过 `pkg/comm/websocket` 订阅呈现。
5. 所有阶段 trace 被 `internal/middleware` 记录（auth/license/logging/error），健康状态由 `api/http/health.go` 汇总暴露。

---

## 七、落地建议

1. **README + style\_guide.md**：作为项目入口规范，应包含命名、示例、prompt 生成约定、常见反模式。
2. **Scaffold 生成脚本**：一键构建上述目录结构与带注释的空模板（shell/Go 可选），确保初始一致性。
3. **CI 校验**：命名规则、schema 变更、prompt 生成输出是否符合 contract（可考虑引入小型 AI linter 做生成内容比对）。
4. **AI Prompt 规范化** 示例：

   > “请创建一个符合 snake\_case 的 `flow_executor.go` 文件，包含注释说明它在 low\_code 中的职责，接收 context 传入 trace metadata 并做好错误包装。”

---

👉 这个版本完全照你原始树状结构渲染，所有路径保留并逐条补齐作用、职责、协作和落地建议。

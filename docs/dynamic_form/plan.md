# DynamicForm（动态表单）设计文档

## 一、概览

DynamicForm 是一个独立的、轻量的动态表单系统，用于 **采集、校验、清洗、展开参数**，主要用途有两类：

1. **Standalone 表单交互**
   直接在单页面/后台操作界面使用，配置字段、验证、条件、默认值，收集用户输入并返回干净的数据或错误信息。适用于运营表单、配置页面、单一工具调用的参数采集等。

2. **为 Agent/Node 提供动态输入**
   DynamicForm 产生的结构化、验证通过的输入可以注入到智能体 (Agent) 的某个 PlanStep 或 Node 里作为执行参数（如 tool 调用参数）。但 DynamicForm **不参与 plan 生成、调度、执行流程**，那部分由 Agent 的 ExecutionPlan/Executor 负责。

核心原则：**职责单一、与 Agent 解耦、输出可直接消费**。

---

## 二、架构与模块划分

```
/pkg/dynamic_form/
  model/         # 表单 schema（field/validation/condition/variables）
  executor/      # 表单执行器：验证/条件/默认/输出清洗（Standalone 用途）
  runtime/       # 运行时上下文（输入、变量、trace，主要在复杂字段依赖中用）
  api/           # 表单相关 HTTP 接口（schema 获取、校验、提交）
  adapter/       # 可选：将表单结果适配成某个 PlanStep/Node 所需的参数格式（仅转换，不做 plan 逻辑）
```

* `model/`：定义字段、验证规则、可见性、默认值、条件表达式、变量模板等。
* `executor/`：执行表单的校验与填充逻辑，输出标准化结果（包括 field-level errors 和 clean inputs）。
* `adapter/`：如果 Agent/Node 有格式或命名差异，做输入字段到目标参数结构的映射。
* `runtime/`：提供上下文用于条件判断、动态默认值展开、变量依赖。
* `api/`：前端/调用方使用的接口，获取 schema、校验 input、提交数据、dry-run（预验证）。

---

## 三、核心数据模型

### 1. 表单字段与验证

```go
type FieldType string

const (
  FieldTypeString  FieldType = "string"
  FieldTypeNumber  FieldType = "number"
  FieldTypeBoolean FieldType = "boolean"
  FieldTypeSelect  FieldType = "select"
  FieldTypeObject  FieldType = "object"
  FieldTypeArray   FieldType = "array"
)

type ValidationRule struct {
  Required     bool
  MinLength    *int
  MaxLength    *int
  Min          *float64
  Max          *float64
  Pattern      string // 正则
  Custom       string // 自定义表达式（比如 CEL/JSONLogic）
  ErrorMessage string
}

type Condition struct {
  Expr map[string]interface{} // 条件表达式，可插入 JSONLogic/CEL/自定义 AST
}

type Field struct {
  Name         string                 // 逻辑键
  Label        string                 // 展示名称
  Type         FieldType              // 类型
  Default      interface{}            // 默认值（可以是表达式）
  Options      []string               // 下拉/枚举选项
  Validations  []ValidationRule       // 验证规则
  Visibility   *Condition             // 是否显示
  Enablement   *Condition             // 是否可编辑
  Description  string
  Extra        map[string]interface{} // 预留扩展
}
```

### 2. 表单 Schema

```go
type FormSchema struct {
  ID          string                 // 表单标识
  Title       string                 // 显示标题
  Description string                 // 说明
  Fields      []Field                // 字段定义
  Variables   map[string]string      // 初始模板变量（可在默认值/条件里引用）
  Metadata    map[string]interface{} // 扩展信息（比如绑定到哪个 PlanStep/Node）
}
```

### 3. 运行时上下文（用于条件/动态默认）

```go
type ExecutionContext struct {
  Inputs    map[string]interface{} // 原始输入
  Variables map[string]interface{} // 评估/展开后的变量（例如 derived 值）
  TraceID   string                 // 追踪 ID
}
```

---

## 四、DynamicForm Executor（表单处理器）

DynamicForm Executor 负责将用户提交的原始数据：

1. 评估可见性/可编辑性（基于 condition）
2. 填充默认值（含表达式求值）
3. 验证字段（ValidationRule）
4. 变量展开（支持模板如 `{{user.name}}`）
5. 输出清洗后的输入和字段级错误

输出格式示例：

```json
{
  "validated_inputs": {
    "customer_id": "C123",
    "tag": "VIP",
    "notify": true
  },
  "field_errors": {
    "customer_id": ["required"],
    "tag": []
  },
  "visible_fields": ["customer_id", "tag", "notify"],
  "enabled_fields": ["customer_id", "tag"],
  "context": {
    "variables": {
      "derived_score": 85
    }
  }
}
```

### 设计原则

* 只处理输入，不做业务执行
* 支持表达式和条件驱动的动态字段行为
* 输出稳定、易映射给上层消费（Agent/Node/工具）

---

## 五、与 Agent/Node 的集成

DynamicForm 本身不参与智能体的计划生成与执行，它的输出**作为参数**被注入某个 PlanStep 或 Node：

1. 上层（调用者）获取 DynamicForm 的 `validated_inputs`。
2. 通过可选的 `adapter` 将这些输入映射为目标 tool/Node 的参数结构（例如转换字段名、嵌套、默认补全）。
3. 将结果填入某个 `contract.PlanStep.Input` 或 Node 的 context 中。
4. Agent 负责把这个 PlanStep 包含在 `ExecutionPlan` 里并最终执行。

例子（伪代码）：

```go
formResult := dynamicformExecutor.Validate(formSchema, rawInput)
if len(formResult.FieldErrors) > 0 {
    return formResult // 直接反馈给 UI
}
stepInput := adapter.MapToApplyTagTool(formResult.ValidatedInputs)
planStep := &contract.PlanStep{
    ID: "apply_tag_step",
    Type: "tool",
    Name: "apply_tag",
    Input: stepInput,
}
plan := &contract.ExecutionPlan{
    Intent: "打标签",
    Steps: []contract.PlanStep{*planStep},
}
agent.Run(ctx, plan.Intent, map[string]interface{}{"plan": plan})
```

---

## 六、API 设计（Standalone 表单场景）

### 1. 获取表单 schema

GET /dynamic\_form/form/{form\_id}
返回表单 schema，包含字段、验证规则、条件、默认值等。

### 2. 验证表单输入

POST /dynamic\_form/form/{form\_id}/validate
请求体：原始输入对象。
返回：

* `validated_inputs`: 清洗后的输入
* `field_errors`: 每个字段的错误列表
* `visible_fields` / `enabled_fields`: 经条件评估的可见/可用
* `context`: 运行时变量（用于展示/调试）

### 3. 提交表单（触发 side-effect 或供上层取值）

POST /dynamic\_form/form/{form\_id}/submit
行为可配置：

* 仅返回清洗后的结果（用于手动构造 plan）
* 触发某个单一 tool
* 返回适配好的参数结构给调用方（agent 预处理）

---

## 七、典型业务场景

### 场景 1：运营后台“升级会员”表单

* 字段：`customer_id`（必填）、`target_level`（枚举）、`notify`（布尔）
* 用户提交后：DynamicForm Executor 验证并返回 `validated_inputs`，再由后端直接调用 `upgrade_membership` tool（不通过 Agent）。

### 场景 2：智能体参数采集 —— “给客户打 VIP 标签并通知”

* 表单收集：`customer_id`, `tag=VIP`, `notify=true`
* DynamicForm 输出清洗输入，adapter 转成 tool 参数，构造 `PlanStep` 填入 `ExecutionPlan`。
* Agent.Run 执行计划，完成标签应用与条件通知。

### 场景 3：表单嵌套默认/派生字段

* 表单定义了一个 `score` 字段，基于 `customer_level` 计算出的默认值（表达式）。
* DynamicForm 在验证前计算并注入 `score`，上层直接拿来做决策或传给 Agent。

---

## 八、扩展建议

1. 表达式引擎可替换：条件、默认、Custom 验证支持 JSONLogic、CEL、自定义 DSL 插件化。
2. 字段级权限：schema 增加可见/编辑基于角色的规则。
3. 模板变量：支持 `{{user.name}}`、`{{previous_step.output}}` 等上下文替换。
4. Input Adapter：提供常用转换器，比如 snake\_case ↔ camelCase、字段合并/拆分、默认填充。
5. 输入快照与回填：前端可缓存最后一次成功输入，支持“修改再跑”体验。

---

## 九、命名与部署建议

* 模块名使用 `dynamic_form` 以区别于 Agent 核心（ExecutionPlan/Flow）。
* DynamicForm 提供的输出标准化后可被多种消费者复用（Agent、独立工具、存储、审核）。
* 建议在表单 schema 中埋 metadata（如绑定哪个 PlanStep、所属业务线），便于追踪。
* 表单服务可以独立部署，Agent 在需要时拉取并消费其输出。

---

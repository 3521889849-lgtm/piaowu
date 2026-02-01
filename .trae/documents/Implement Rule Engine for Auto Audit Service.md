# Implementation Plan: Distributed Audit System - Rule Engine

This plan focuses on the first phase of the distributed audit system: implementing the **Rule Engine** for the Auto Audit Service.

## 1. Database Model Design
Create the `RuleConfig` model to store audit rules.
- **File**: `common/model/audit/rule_config.go`
- **Schema**:
  - `ID`: Primary Key
  - `BizType`: Business type (e.g., ticket_order, merchant_apply)
  - `RuleName`: Descriptive name
  - `Expression`: JSON string defining the rule logic (e.g., `{"field":"amount", "op":">", "val":1000}`)
  - `Action`: Result if matched (Pass, Reject, Review)
  - `Priority`: Execution order
  - `Status`: Enabled/Disabled
- **Migration**: Update `common/db/mysql.go` to include `RuleConfig` in `AutoMigrate`.

## 2. Rule Engine Implementation
Implement the core logic to load and execute rules.
- **Directory**: `rpc/audit/component/rule_engine/`
- **Components**:
  - **`types.go`**: Define input/output structures and the Rule definition interface.
  - **`loader.go`**: Logic to load rules from MySQL and cache them in Redis (with periodic refresh).
  - **`executor.go`**: The evaluation logic. It will support:
    - Basic operators: `>`, `<`, `=`, `IN`
    - Logic operators: `AND`, `OR` (via rule priority and short-circuiting)
    - Action handling: Return `Pass`, `Reject`, or `Review`

## 3. Integration with Auto Audit Service
Refactor the existing `AutoAudit` service to use the new Rule Engine.
- **File**: `rpc/audit/service/auto_audit.go`
- **Changes**:
  - Replace the hardcoded `switch` logic with `RuleEngine.Execute(ctx, input)`.
  - Map the `ApplyAuditReq` to the Rule Engine's input format.

## 4. Verification
- **Unit Test**: Create `rpc/audit/component/rule_engine/engine_test.go` to verify rule evaluation logic.
- **Integration Test**: Verify the full flow via the existing `AutoAudit` RPC interface.

This implementation lays the foundation for dynamic rule configuration without code changes.
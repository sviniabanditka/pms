# Proto-контракты: команды, Filter AST, sync

Code-ready protobuf (proto3) под ConnectRPC (ADR-0003 S1). buf-кодген → Go-типы + TS-клиент.
Целевой путь при скаффолде — `proto/pms/v1/`. Идемпотентность — заголовок `Idempotency-Key`
(= `command.id`, UUIDv7), не поле сообщения. Связано: [commands](commands.md), [query-engine](query-engine.md), [sync-protocol](sync-protocol.md).

---

## common.proto

```proto
syntax = "proto3";
package pms.v1;

// Результат любой write-команды.
message CommandResult {
  string entity_id   = 1;
  int64  new_version = 2;
}

// Скоуп выборки/представления.
message Scope {
  enum Kind { WORKSPACE = 0; TEAM = 1; PROJECT = 2; PERSONAL = 3; }
  Kind   kind       = 1;
  string team_id    = 2; // при TEAM
  string project_id = 3; // при PROJECT
}
```

---

## filter.proto (Filter AST §6.2)

```proto
syntax = "proto3";
package pms.v1;

message FilterNode {
  oneof node {
    BoolGroup  and_      = 1;
    BoolGroup  or_       = 2;
    FilterNode not_      = 3;
    Predicate  predicate = 4;
  }
}

message BoolGroup { repeated FilterNode children = 1; }

message Predicate {
  FieldRef field = 1;
  Operator op    = 2;
  FilterValue value = 3;
}

message FieldRef {
  oneof ref {
    string builtin         = 1; // 'status' | 'assignee' | 'state.category' | ...
    string custom_field_id = 2;
  }
}

enum Operator {
  OP_UNSPECIFIED = 0;
  EQ = 1;  NEQ = 2;  IN = 3;  NIN = 4;
  LT = 5;  LTE = 6;  GT = 7;  GTE = 8;
  CONTAINS = 9; STARTS_WITH = 10;
  IS_SET = 11; IS_NOT_SET = 12;
  BEFORE = 13; AFTER = 14; WITHIN_NEXT = 15; WITHIN_PAST = 16;
}

message RelativeDate {
  enum Unit { DAY = 0; WEEK = 1; MONTH = 2; }
  Unit  unit   = 1;
  int32 amount = 2;
}

message FilterValue {
  oneof v {
    string       s        = 1;
    double       n        = 2;
    bool         b        = 3;
    StringList   list     = 4;  // для IN/NIN
    RelativeDate relative = 5;  // для WITHIN_*
    bool         null_v   = 6;  // маркер null (IS_SET и т.п. — без value)
  }
}
message StringList { repeated string values = 1; }
```

---

## view.proto (ViewDefinition §6.1)

```proto
syntax = "proto3";
package pms.v1;
import "pms/v1/filter.proto";
import "pms/v1/common.proto";

message GroupSpec { FieldRef field = 1; }
message OrderSpec { FieldRef field = 1; bool desc = 2; bool manual = 3; }

message ViewDefinition {
  string     data_source        = 1; // 'issue' | 'doc' | ...
  Scope      scope              = 2;
  FilterNode filter             = 3;
  GroupSpec  grouping           = 4;
  GroupSpec  sub_grouping       = 5;
  repeated OrderSpec ordering    = 6;
  string     layout             = 7; // 'board'|'list'|'table'|'calendar'
  bytes      layout_options     = 8; // json
  repeated string display_props = 9;
  bytes      options            = 10; // json: rollupDescendants, showSubIssues, showCompleted, density
}
```

---

## issue_service.proto (репрезентативно; остальные сервисы — по тому же образцу)

```proto
syntax = "proto3";
package pms.v1;
import "pms/v1/common.proto";

service IssueService {
  rpc CreateIssue       (CreateIssueRequest)       returns (CommandResult);
  rpc UpdateIssue       (UpdateIssueRequest)       returns (CommandResult); // частичный набор полей
  rpc UpdateIssueState  (UpdateIssueStateRequest)  returns (CommandResult); // ставит started/completed
  rpc AssignIssue       (AssignIssueRequest)       returns (CommandResult);
  rpc ReorderIssue      (ReorderIssueRequest)      returns (CommandResult);
  rpc AddLabel          (LabelRequest)             returns (CommandResult);
  rpc RemoveLabel       (LabelRequest)             returns (CommandResult);
  rpc AddRelation       (RelationRequest)          returns (CommandResult);
  rpc SetCustomFieldValue (CustomFieldValueRequest) returns (CommandResult);
  rpc ArchiveIssue      (EntityRef)                returns (CommandResult);
  rpc DeleteIssue       (EntityRef)                returns (CommandResult);
  rpc BulkUpdateIssues  (BulkUpdateRequest)        returns (CommandResult);
  rpc AddComment        (AddCommentRequest)        returns (CommandResult);
}

message EntityRef { string workspace_id = 1; string id = 2; }

message CreateIssueRequest {
  string workspace_id = 1;
  string id           = 2; // client-generated UUIDv7 (== Idempotency-Key)
  string team_id      = 3;
  string title        = 4;
  bytes  description  = 5; // TipTap JSON
  string state_id     = 6;
  int32  priority     = 7;
  string assignee_id  = 8;
  string project_id   = 9;
  string cycle_id     = 10;
  string parent_id    = 11; // sub-issue
  repeated string label_ids = 12;
}

message UpdateIssueRequest {
  string workspace_id = 1;
  string id           = 2;
  int64  base_version = 3;          // per-field LWW (§sync-protocol 6)
  // только присутствующие поля применяются:
  optional string title       = 4;
  optional bytes  description = 5;
  optional int32  priority    = 6;
  optional string assignee_id = 7;
  optional string project_id  = 8;
  optional string cycle_id    = 9;
  optional string due_date    = 10; // ISO date
  optional string start_date  = 11;
  optional string scheduled_date = 12;
  optional string estimate    = 13;
}

message UpdateIssueStateRequest {
  string workspace_id = 1; string id = 2; int64 base_version = 3; string state_id = 4;
}
message AssignIssueRequest  { string workspace_id = 1; string id = 2; int64 base_version = 3; string assignee_id = 4; }
message ReorderIssueRequest { string workspace_id = 1; string id = 2; string context = 3; string before_id = 4; string after_id = 5; }
message LabelRequest        { string workspace_id = 1; string issue_id = 2; string label_id = 3; }
message RelationRequest     { string workspace_id = 1; string source_id = 2; string target_id = 3; string type = 4; }
message CustomFieldValueRequest { string workspace_id = 1; string issue_id = 2; string field_id = 3; bytes value = 4; }
message BulkUpdateRequest   { string workspace_id = 1; repeated string ids = 2; bytes patch = 3; }
message AddCommentRequest   { string workspace_id = 1; string id = 2; string entity_type = 3; string entity_id = 4; bytes body = 5; repeated string mentions = 6; }
```

> Прочие сервисы (`TeamService`, `PlanningService`, `ViewService`, `WorkspaceService`, `DocService`)
> определяются по тому же шаблону: команда → message-запрос с `workspace_id` + client-`id` + `base_version`
> (где есть LWW) → `CommandResult`. Полный каталог команд — [commands](commands.md) §2.

---

## query_service.proto (чтение/метрики)

```proto
service QueryService {
  rpc RunView (RunViewRequest) returns (RunViewResponse); // исполнить ViewDefinition на сервере
  rpc Metric  (MetricRequest)  returns (MetricResponse);  // count/sum/avg поверх Filter AST
}

message RunViewRequest  { string workspace_id = 1; ViewDefinition def = 2; bytes page_cursor = 3; }
message RunViewResponse { bytes grouped_result = 1; bytes next_cursor = 2; } // grouped_result — json
message MetricRequest   { string workspace_id = 1; MetricSpec spec = 2; }
message MetricResponse  { double value = 1; }
message MetricSpec      { string agg = 1; FieldRef field = 2; FilterNode filter = 3; }
```

---

## sync_service.proto (bootstrap + догон; live-дельты — по WS, см. ws-protocol)

```proto
service SyncService {
  rpc Bootstrap (BootstrapRequest) returns (stream BootstrapChunk); // прогрессивный срез
  rpc Changes   (ChangesRequest)   returns (ChangesResponse);       // догон since=cursor
}

message BootstrapRequest { string workspace_id = 1; repeated string team_ids = 2; }
message BootstrapChunk   { string entity_type = 1; bytes entities = 2; int64 cursor = 3; } // entities — json batch
message ChangesRequest   { string workspace_id = 1; int64 since_cursor = 2; }
message ChangesResponse  { repeated Delta deltas = 1; int64 cursor = 2; }

message Delta {
  int64  seq            = 1;
  string entity_type    = 2;
  string entity_id      = 3;
  int64  version        = 4;
  string op             = 5; // create|update|delete|archive
  bytes  changed_fields = 6; // json
}
```

## Замечания
- `bytes ... // json` для нестабильных/произвольных структур (TipTap, layout_options, grouped_result) — proto-схема не сковывает rich-контент; стабильные поля — типизированы.
- `optional` (proto3 field presence) на полях `UpdateIssueRequest` → отличаем «не менять» от «обнулить».
- buf-линт + breaking-check в CI; эволюция контрактов — только обратимо-совместимо.

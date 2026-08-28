## action

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| seqid       | int             | NULL           | NO          |            |                |                |
| state       | tinyint         | NULL           | NO          |            |                |                |
| topic       | char(36)        | NULL           | NO          |            |                |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |
| value       | varchar(256)    | NULL           | NO          |            |                |                |

## behavior

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| count       | int             | NULL           | NO          |            |                |                |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| extra       | json            | NULL           | YES         |            |                |                |
| flag        | varchar(100)    | NULL           | NO          | MUL        |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |

## bots

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| created_at  | datetime    | NULL           | NO          |            |                   |                |
| id          | bigint      | NULL           | NO          | PRI        | auto_increment    |                |
| name        | varchar(50) | NULL           | NO          |            |                   |                |
| state       | tinyint     | 0              | NO          |            | DEFAULT_GENERATED |                |
| updated_at  | datetime    | NULL           | NO          |            |                   |                |

## channels

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime    | NULL           | NO          |            |                |                |
| flag        | varchar(36) | NULL           | NO          | MUL        |                |                |
| id          | bigint      | NULL           | NO          | PRI        | auto_increment |                |
| name        | varchar(50) | NULL           | NO          |            |                |                |
| state       | tinyint     | 0              | NO          |            |                |                |
| updated_at  | datetime    | NULL           | NO          |            |                |                |

## configs

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| key         | varchar(100)    | NULL           | NO          |            |                |                |
| topic       | char(36)        | NULL           | NO          |            |                |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |
| value       | json            | NULL           | NO          |            |                |                |

## counter_records

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| counter_id  | bigint unsigned | 0              | NO          | PRI        | DEFAULT_GENERATED |                |
| created_at  | datetime        | NULL           | NO          |            |                   |                |
| digit       | int             | NULL           | NO          |            |                   |                |

## counters

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| digit       | bigint          | NULL           | NO          |            |                |                |
| flag        | varchar(100)    | NULL           | NO          |            |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| status      | int             | NULL           | NO          |            |                |                |
| topic       | char(36)        | NULL           | NO          |            |                |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |

## cycles

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime    | NULL           | NO          |            |                |                |
| end_date    | date        | NULL           | NO          |            |                |                |
| id          | bigint      | NULL           | NO          | PRI        | auto_increment |                |
| objectives  | json        | NULL           | NO          |            |                |                |
| start_date  | date        | NULL           | NO          |            |                |                |
| state       | tinyint     | NULL           | NO          |            |                |                |
| topic       | char(36)    | NULL           | NO          |            |                |                |
| uid         | char(36)    | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime    | NULL           | NO          |            |                |                |

## dag

| COLUMN_NAME    | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| -------------- | --------------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| created_at     | datetime        | NULL           | NO          |            |                   |                |
| edges          | json            | NULL           | NO          |            |                   |                |
| id             | bigint unsigned | NULL           | NO          | PRI        | auto_increment    |                |
| nodes          | json            | NULL           | NO          |            |                   |                |
| script_id      | bigint          | NULL           | NO          |            |                   |                |
| script_version | smallint        | NULL           | NO          |            |                   |                |
| updated_at     | datetime        | NULL           | NO          |            |                   |                |
| workflow_id    | bigint          | 0              | NO          | MUL        | DEFAULT_GENERATED |                |

## data

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| key         | varchar(100)    | NULL           | NO          |            |                |                |
| topic       | char(36)        | NULL           | NO          |            |                |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |
| value       | json            | NULL           | NO          |            |                |                |

## fileuploads

| COLUMN_NAME | COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ------------- | -------------- | ----------- | ---------- | ----- | -------------- |
| created_at  | datetime      | NULL           | NO          |            |       |                |
| id          | bigint        | NULL           | NO          | PRI        |       |                |
| location    | varchar(2048) | NULL           | NO          |            |       |                |
| mimetype    | varchar(255)  | NULL           | NO          |            |       |                |
| name        | varchar(255)  | NULL           | NO          |            |       |                |
| size        | bigint        | NULL           | NO          |            |       |                |
| state       | int           | NULL           | NO          | MUL        |       |                |
| uid         | char(36)      | NULL           | NO          | MUL        |       |                |
| updated_at  | datetime      | NULL           | NO          |            |       |                |

## form

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| extra       | json            | NULL           | YES         |            |                |                |
| form_id     | varchar(100)    | NULL           | NO          | MUL        |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| schema      | json            | NULL           | NO          |            |                |                |
| state       | tinyint         | NULL           | NO          |            |                |                |
| topic       | char(36)        | NULL           | NO          |            |                |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |
| values      | json            | NULL           | YES         |            |                |                |

## instruct

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| bot         | varchar(50)     | NULL           | NO          |            |                |                |
| content     | json            | NULL           | NO          |            |                |                |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| expire_at   | datetime        | NULL           | NO          |            |                |                |
| flag        | varchar(50)     | NULL           | NO          |            |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| no          | char(25)        | NULL           | NO          | MUL        |                |                |
| object      | varchar(20)     | NULL           | NO          |            |                |                |
| priority    | int             | NULL           | NO          |            |                |                |
| state       | tinyint         | NULL           | NO          |            |                |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |

## jobs

| COLUMN_NAME    | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| -------------- | --------------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| created_at     | datetime        | NULL           | NO          |            |                   |                |
| dag_id         | bigint          | 0              | NO          |            | DEFAULT_GENERATED |                |
| ended_at       | datetime        | NULL           | YES         |            |                   |                |
| id             | bigint unsigned | NULL           | NO          | PRI        | auto_increment    |                |
| script_version | smallint        | 0              | NO          |            | DEFAULT_GENERATED |                |
| started_at     | datetime        | NULL           | YES         |            |                   |                |
| state          | tinyint         | NULL           | NO          | MUL        |                   |                |
| topic          | char(36)        | NULL           | NO          |            |                   |                |
| trigger_id     | bigint          | 0              | NO          |            | DEFAULT_GENERATED |                |
| uid            | char(36)        | NULL           | NO          | MUL        |                   |                |
| updated_at     | datetime        | NULL           | NO          |            |                   |                |
| workflow_id    | bigint          | 0              | NO          | MUL        | DEFAULT_GENERATED |                |

## key_result_values

| COLUMN_NAME   | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ------------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at    | datetime        | NULL           | NO          |            |                |                |
| id            | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| key_result_id | bigint          | NULL           | YES         | MUL        |                |                |
| memo          | varchar(1000)   |                | NO          |            |                |                |
| updated_at    | datetime        | NULL           | NO          |            |                |                |
| value         | int             | NULL           | NO          |            |                |                |

## key_results

| COLUMN_NAME   | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| ------------- | --------------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| created_at    | datetime        | NULL           | NO          |            |                   |                |
| current_value | int             | NULL           | NO          |            |                   |                |
| id            | bigint unsigned | NULL           | NO          | PRI        | auto_increment    |                |
| initial_value | int             | NULL           | NO          |            |                   |                |
| memo          | varchar(1000)   | NULL           | NO          |            |                   |                |
| objective_id  | bigint          | 0              | NO          |            | DEFAULT_GENERATED |                |
| sequence      | int             | NULL           | NO          |            |                   |                |
| tag           | varchar(100)    | NULL           | NO          |            |                   |                |
| target_value  | int             | NULL           | NO          |            |                   |                |
| title         | varchar(100)    | NULL           | NO          |            |                   |                |
| topic         | char(36)        | NULL           | NO          |            |                   |                |
| uid           | char(36)        | NULL           | NO          | MUL        |                   |                |
| updated_at    | datetime        | NULL           | NO          |            |                   |                |
| value_mode    | varchar(20)     |                | NO          |            |                   |                |

## messages

| COLUMN_NAME     | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| --------------- | ----------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| content         | json        | NULL           | YES         |            |                   |                |
| created_at      | datetime    | NULL           | NO          |            |                   |                |
| deleted_at      | datetime    | NULL           | YES         |            |                   |                |
| flag            | char(36)    | NULL           | NO          | UNI        |                   |                |
| id              | bigint      | NULL           | NO          | PRI        | auto_increment    |                |
| platform_id     | bigint      | 0              | NO          | MUL        | DEFAULT_GENERATED |                |
| platform_msg_id | varchar(50) |                | NO          |            |                   |                |
| state           | tinyint     | NULL           | NO          |            |                   |                |
| topic           | char(36)    | NULL           | NO          | MUL        |                   |                |
| updated_at      | datetime    | NULL           | NO          |            |                   |                |

## oauth

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| extra       | json            | NULL           | NO          |            |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| name        | varchar(100)    | NULL           | NO          |            |                |                |
| token       | varchar(256)    | NULL           | NO          |            |                |                |
| topic       | char(36)        | NULL           | NO          |            |                |                |
| type        | varchar(50)     | NULL           | NO          |            |                |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |

## objectives

| COLUMN_NAME   | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ------------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_data  | datetime        | NULL           | NO          |            |                |                |
| current_value | int             | NULL           | NO          |            |                |                |
| feasibility   | varchar(1000)   | NULL           | NO          |            |                |                |
| id            | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| is_plan       | tinyint         | 0              | NO          |            |                |                |
| memo          | varchar(1000)   | NULL           | NO          |            |                |                |
| motive        | varchar(1000)   | NULL           | NO          |            |                |                |
| plan_end      | date            | NULL           | NO          |            |                |                |
| plan_start    | date            | NULL           | NO          |            |                |                |
| progress      | tinyint         | 0              | NO          |            |                |                |
| sequence      | int             | NULL           | NO          |            |                |                |
| tag           | varchar(100)    | NULL           | NO          |            |                |                |
| title         | varchar(100)    | NULL           | NO          |            |                |                |
| topic         | char(36)        | NULL           | NO          |            |                |                |
| total_value   | int             | NULL           | NO          |            |                |                |
| uid           | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_date  | datetime        | NULL           | NO          |            |                |                |

## pages

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| page_id     | varchar(100)    | NULL           | NO          | MUL        |                |                |
| schema      | json            | NULL           | NO          |            |                |                |
| state       | tinyint         | NULL           | NO          |            |                |                |
| topic       | char(36)        | NULL           | NO          |            |                |                |
| type        | varchar(100)    | NULL           | NO          |            |                |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |

## parameter

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| expired_at  | datetime        | NULL           | NO          |            |                |                |
| flag        | char(25)        | NULL           | NO          | UNI        |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| params      | json            | NULL           | YES         |            |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |

## pipelines

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| flag        | char(25)        | NULL           | NO          |            |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| rule_id     | varchar(100)    | NULL           | NO          |            |                |                |
| stage       | int             | NULL           | NO          |            |                |                |
| state       | tinyint         | NULL           | NO          |            |                |                |
| topic       | char(36)        | NULL           | NO          |            |                |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |
| values      | json            | NULL           | YES         |            |                |                |
| version     | int             | NULL           | NO          |            |                |                |

## platform_channel_users

| COLUMN_NAME  | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ------------ | ----------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| channel_flag | varchar(50) | NULL           | NO          | MUL        |                |                |
| created_at   | datetime    | NULL           | NO          |            |                |                |
| id           | bigint      | NULL           | NO          | PRI        | auto_increment |                |
| platform_id  | bigint      | 0              | NO          | MUL        |                |                |
| updated_at   | datetime    | NULL           | NO          |            |                |                |
| user_flag    | varchar(50) | NULL           | NO          | MUL        |                |                |

## platform_channels

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| channel_id  | bigint      | 0              | NO          | MUL        |                |                |
| created_at  | datetime    | NULL           | NO          |            |                |                |
| flag        | varchar(50) | 0              | NO          |            |                |                |
| id          | bigint      | NULL           | NO          | PRI        | auto_increment |                |
| platform_id | bigint      | 0              | NO          | MUL        |                |                |
| updated_at  | datetime    | NULL           | NO          |            |                |                |

## platform_users

| COLUMN_NAME | COLUMN_TYPE  | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | ------------ | -------------- | ----------- | ---------- | -------------- | -------------- |
| avatar_url  | varchar(200) | NULL           | NO          |            |                |                |
| created_at  | datetime     | NULL           | NO          |            |                |                |
| email       | varchar(50)  | NULL           | NO          |            |                |                |
| flag        | varchar(36)  | NULL           | NO          | MUL        |                |                |
| id          | bigint       | NULL           | NO          | PRI        | auto_increment |                |
| is_bot      | tinyint(1)   | 0              | NO          |            |                |                |
| name        | varchar(30)  | NULL           | NO          |            |                |                |
| platform_id | bigint       | 0              | NO          | MUL        |                |                |
| updated_at  | datetime     | NULL           | NO          |            |                |                |
| user_id     | bigint       | 0              | NO          | MUL        |                |                |

## platforms

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime    | NULL           | NO          |            |                |                |
| id          | bigint      | NULL           | NO          | PRI        | auto_increment |                |
| name        | varchar(50) | NULL           | NO          |            |                |                |
| updated_at  | datetime    | NULL           | NO          |            |                |                |

## review_evaluations

| COLUMN_NAME | COLUMN_TYPE  | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| ----------- | ------------ | -------------- | ----------- | ---------- | ----------------- | -------------- |
| created_at  | datetime     | NULL           | NO          |            |                   |                |
| id          | bigint       | NULL           | NO          | PRI        | auto_increment    |                |
| question    | varchar(255) | NULL           | NO          |            |                   |                |
| reason      | varchar(255) | NULL           | NO          |            |                   |                |
| review_id   | bigint       | 0              | NO          | MUL        | DEFAULT_GENERATED |                |
| solving     | varchar(255) | NULL           | NO          |            |                   |                |
| topic       | char(36)     | NULL           | NO          |            |                   |                |
| uid         | char(36)     | NULL           | NO          | MUL        |                   |                |
| updated_at  | datetime     | NULL           | NO          |            |                   |                |

## reviews

| COLUMN_NAME  | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| ------------ | ----------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| created_at   | datetime    | NULL           | NO          |            |                   |                |
| id           | bigint      | NULL           | NO          | PRI        | auto_increment    |                |
| objective_id | bigint      | 0              | NO          |            | DEFAULT_GENERATED |                |
| rating       | tinyint     | NULL           | NO          |            |                   |                |
| topic        | char(36)    | NULL           | NO          |            |                   |                |
| type         | tinyint     | NULL           | NO          |            |                   |                |
| uid          | char(36)    | NULL           | NO          | MUL        |                   |                |
| updated_at   | datetime    | NULL           | NO          |            |                   |                |

## schema_migrations

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| dirty       | tinyint     | 0              | NO          |            | DEFAULT_GENERATED |                |
| version     | int         | NULL           | NO          | PRI        | auto_increment    |                |

## session

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| init        | json            | NULL           | NO          |            |                |                |
| rule_id     | varchar(100)    | NULL           | NO          |            |                |                |
| state       | tinyint         | NULL           | NO          |            |                |                |
| topic       | char(36)        | NULL           | NO          |            |                |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |
| values      | json            | NULL           | NO          |            |                |                |

## steps

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| action      | json            | NULL           | NO          |            |                   |                |
| created_at  | datetime        | NULL           | NO          |            |                   |                |
| depend      | json            | NULL           | YES         |            |                   |                |
| describe    | varchar(300)    |                | NO          |            |                   |                |
| ended_at    | datetime        | NULL           | YES         |            |                   |                |
| error       | varchar(1000)   | NULL           | YES         |            |                   |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment    |                |
| input       | json            | NULL           | YES         |            |                   |                |
| job_id      | bigint          | 0              | NO          | MUL        | DEFAULT_GENERATED |                |
| name        | varchar(100)    |                | NO          |            |                   |                |
| node_id     | varchar(50)     |                | NO          | MUL        |                   |                |
| output      | json            | NULL           | YES         |            |                   |                |
| started_at  | datetime        | NULL           | YES         |            |                   |                |
| state       | tinyint         | NULL           | NO          | MUL        |                   |                |
| topic       | char(36)        | NULL           | NO          |            |                   |                |
| uid         | char(36)        | NULL           | NO          | MUL        |                   |                |
| updated_at  | datetime        | NULL           | NO          |            |                   |                |

## todos

| COLUMN_NAME       | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| ----------------- | --------------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| category          | varchar(100)    | NULL           | NO          |            |                   |                |
| complete          | tinyint         | NULL           | NO          |            |                   |                |
| content           | varchar(1000)   | NULL           | NO          |            |                   |                |
| created_at        | datetime        | NULL           | NO          |            |                   |                |
| id                | bigint unsigned | NULL           | NO          | PRI        | auto_increment    |                |
| is_remind_at_time | tinyint         | NULL           | NO          |            |                   |                |
| key_result_id     | bigint          | 0              | NO          |            | DEFAULT_GENERATED |                |
| parent_id         | bigint          | 0              | NO          | MUL        | DEFAULT_GENERATED |                |
| priority          | int             | NULL           | NO          |            |                   |                |
| remark            | varchar(100)    | NULL           | NO          |            |                   |                |
| remind_at         | bigint          | NULL           | NO          |            |                   |                |
| repeat_end_at     | bigint          | NULL           | NO          |            |                   |                |
| repeat_method     | varchar(100)    | NULL           | NO          |            |                   |                |
| repeat_rule       | varchar(100)    | NULL           | NO          |            |                   |                |
| sequence          | int             | NULL           | NO          |            |                   |                |
| topic             | char(36)        | NULL           | NO          |            |                   |                |
| uid               | char(36)        | NULL           | NO          | MUL        |                   |                |
| updated_at        | datetime        | NULL           | NO          |            |                   |                |

## users

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| flag        | char(36)        | NULL           | NO          | UNI        |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| name        | varchar(50)     | NULL           | NO          |            |                |                |
| state       | smallint        | 0              | NO          |            |                |                |
| tags        | json            | NULL           | YES         |            |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |

## workflow

| COLUMN_NAME      | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ---------------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| canceled_count   | int             | 0              | NO          |            |                |                |
| created_at       | datetime        | NULL           | NO          |            |                |                |
| describe         | varchar(300)    | NULL           | NO          |            |                |                |
| failed_count     | int             | 0              | NO          |            |                |                |
| flag             | char(25)        | NULL           | NO          | MUL        |                |                |
| id               | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| name             | varchar(100)    | NULL           | NO          |            |                |                |
| running_count    | int             | 0              | NO          |            |                |                |
| state            | tinyint         | NULL           | NO          |            |                |                |
| successful_count | int             | 0              | NO          |            |                |                |
| topic            | char(36)        | NULL           | NO          |            |                |                |
| uid              | char(36)        | NULL           | NO          | MUL        |                |                |
| updated_at       | datetime        | NULL           | NO          |            |                |                |

## workflow_script

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA          | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | -------------- | -------------- |
| code        | text            | NULL           | NO          |            |                |                |
| created_at  | datetime        | NULL           | NO          |            |                |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment |                |
| lang        | varchar(10)     | NULL           | NO          |            |                |                |
| updated_at  | datetime        | NULL           | NO          |            |                |                |
| version     | smallint        | 1              | NO          |            |                |                |
| workflow_id | bigint unsigned | NULL           | NO          |            |                |                |

## workflow_trigger

| COLUMN_NAME | COLUMN_TYPE     | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA             | COLUMN_COMMENT |
| ----------- | --------------- | -------------- | ----------- | ---------- | ----------------- | -------------- |
| count       | int             | 0              | NO          |            |                   |                |
| created_at  | datetime        | NULL           | NO          |            |                   |                |
| id          | bigint unsigned | NULL           | NO          | PRI        | auto_increment    |                |
| rule        | json            | NULL           | YES         |            |                   |                |
| state       | tinyint         | NULL           | NO          |            |                   |                |
| type        | varchar(20)     | NULL           | NO          |            |                   |                |
| updated_at  | datetime        | NULL           | NO          |            |                   |                |
| workflow_id | bigint          | 0              | NO          | MUL        | DEFAULT_GENERATED |                |

## life_profiles

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| flag | varchar | NULL | NO | UNI | | |
| user_id | varchar | NULL | NO | UNI | | |
| nickname | varchar | | NO | | | |
| level | int | 1 | NO | | | |
| exp | bigint | 0 | NO | | | |
| gold | int | 0 | NO | | | |
| class_type | varchar | Architect | NO | | | |
| base_drop_rate_bonus | double | 0 | NO | | | |
| pity_by_tier | json | NULL | YES | | | |
| created_at | datetime | NULL | NO | | | |
| updated_at | datetime | NULL | NO | | | |

## life_characteristics

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| flag | varchar | NULL | NO | UNI | | |
| life_profile_id | bigint | NULL | NO | MUL | | |
| code | varchar | NULL | NO | | | |
| name | varchar | NULL | NO | | | |
| level | int | 1 | NO | | | |
| current_exp | bigint | 0 | NO | | | |
| created_at | datetime | NULL | NO | | | |
| updated_at | datetime | NULL | NO | | | |

## life_skills

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| flag | varchar | NULL | NO | UNI | | |
| life_profile_id | bigint | NULL | NO | MUL | | |
| characteristic_id | bigint | NULL | NO | | | |
| name | varchar | NULL | NO | | | |
| level | int | 1 | NO | | | |
| current_exp | bigint | 0 | NO | | | |
| exp_to_characteristic_ratio | double | 0.5 | NO | | | |
| created_at | datetime | NULL | NO | | | |
| updated_at | datetime | NULL | NO | | | |

## life_goals

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| flag | varchar | NULL | NO | UNI | | |
| life_profile_id | bigint | NULL | NO | MUL | | |
| title | varchar | NULL | NO | | | |
| category | varchar | NULL | NO | | | |
| status | varchar | Active | NO | | | |
| created_at | datetime | NULL | NO | | | |
| updated_at | datetime | NULL | NO | | | |

## life_quests

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| flag | varchar | NULL | NO | UNI | | |
| life_profile_id | bigint | NULL | NO | MUL | | |
| goal_id | bigint | NULL | YES | | | |
| skill_id | bigint | NULL | NO | MUL | | |
| title | varchar | NULL | NO | | | |
| prompt | text | | NO | | | |
| type | varchar | One-Time | NO | | | |
| ai_evaluated_difficulty | varchar | E | NO | | | |
| ai_evaluated_fear | int | 1 | NO | | | |
| base_exp_reward | int | 10 | NO | | | |
| base_gold_reward | int | 5 | NO | | | |
| drop_tier | varchar | Common | NO | | | |
| status | varchar | Pending | NO | | | |
| created_at | datetime | NULL | NO | | | |
| updated_at | datetime | NULL | NO | | | |
| completed_at | datetime | NULL | YES | | | |

## life_ai_contexts

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| life_profile_id | bigint | NULL | NO | UNI | | |
| ai_dm_personality | text | | NO | | | |
| historical_completion_rate | double | 0 | NO | | | |
| recent_mood_and_burnout | json | NULL | YES | | | |
| updated_at | datetime | NULL | NO | | | |

## life_equipments

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| flag | varchar | NULL | NO | UNI | | |
| name | varchar | NULL | NO | | | |
| rarity | varchar | Common | NO | MUL | | |
| slot_type | varchar | NULL | NO | MUL | | |
| stat_buffs | json | NULL | YES | | | |
| ai_unlocked_privilege | json | NULL | YES | | | |
| ai_lore_text | text | | NO | | | |
| created_at | datetime | NULL | NO | | | |

## life_inventories

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| flag | varchar | NULL | NO | UNI | | |
| life_profile_id | bigint | NULL | NO | MUL | | |
| equipment_id | bigint | NULL | NO | MUL | | |
| source_quest_id | bigint | NULL | YES | | | |
| instance_name | varchar | | NO | | | |
| instance_lore | text | | NO | | | |
| instance_buffs | json | NULL | YES | | | |
| lore_status | varchar | none | NO | MUL | | |
| tarnished_until | datetime | NULL | YES | | | |
| acquired_at | datetime | NULL | NO | | | |

## life_equipped_slots

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| life_profile_id | bigint | NULL | NO | UNI | | |
| head_slot | bigint | NULL | YES | | | |
| weapon_slot | bigint | NULL | YES | | | |
| armor_slot | bigint | NULL | YES | | | |
| shoes_slot | bigint | NULL | YES | | | |
| accessory_slot | bigint | NULL | YES | | | |
| artifact_slot | bigint | NULL | YES | | | |
| tarnished_until | datetime | NULL | YES | | | |
| updated_at | datetime | NULL | NO | | | |

## life_loot_tables

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| drop_tier | varchar | NULL | NO | UNI | | |
| base_drop_chance | double | NULL | NO | | | |
| item_pool_flags | json | NULL | YES | | | |
| updated_at | datetime | NULL | NO | | | |

## life_action_logs

| COLUMN_NAME | COLUMN_TYPE | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |
| ----------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |
| id | bigint | NULL | NO | PRI | | |
| flag | varchar | NULL | NO | UNI | | |
| life_profile_id | bigint | NULL | NO | MUL | | |
| quest_id | bigint | NULL | NO | | | |
| gained_exp | int | NULL | NO | | | |
| gained_gold | int | NULL | NO | | | |
| dropped_inventory_id | bigint | NULL | YES | | | |
| dice_roll_result | double | NULL | YES | | | |
| created_at | datetime | NULL | NO | | | |

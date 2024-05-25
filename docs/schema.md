## behavior 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| flag | varchar(100) | NULL | NO | MUL |  |  |
| count | int | NULL | NO |  |  |  |
| extra | json | NULL | YES |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## bots 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| name | varchar(50) | NULL | NO |  |  |  |
| state | tinyint | 0 | NO |  | DEFAULT_GENERATED |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## channels 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| name | varchar(50) | NULL | NO |  |  |  |
| flag | varchar(36) | NULL | NO | MUL |  |  |
| state | tinyint | 0 | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## configs 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| key | varchar(100) | NULL | NO |  |  |  |
| value | json | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## counter_records 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| counter_id | bigint unsigned | 0 | NO | PRI | DEFAULT_GENERATED |  |
| digit | int | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |


## counters 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| flag | varchar(100) | NULL | NO |  |  |  |
| digit | bigint | NULL | NO |  |  |  |
| status | int | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## cycles 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| objectives | json | NULL | NO |  |  |  |
| start_date | date | NULL | NO |  |  |  |
| end_date | date | NULL | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## dag 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| workflow_id | bigint | 0 | NO | MUL | DEFAULT_GENERATED |  |
| script_id | bigint | NULL | NO |  |  |  |
| script_version | smallint | NULL | NO |  |  |  |
| nodes | json | NULL | NO |  |  |  |
| edges | json | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## data 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| key | varchar(100) | NULL | NO |  |  |  |
| value | json | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## fileuploads 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| name | varchar(255) | NULL | NO |  |  |  |
| mimetype | varchar(255) | NULL | NO |  |  |  |
| size | bigint | NULL | NO |  |  |  |
| location | varchar(2048) | NULL | NO |  |  |  |
| state | int | NULL | NO | MUL |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## form 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| form_id | varchar(100) | NULL | NO | MUL |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| schema | json | NULL | NO |  |  |  |
| values | json | NULL | YES |  |  |  |
| extra | json | NULL | YES |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## instruct 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| no | char(25) | NULL | NO | MUL |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| object | varchar(20) | NULL | NO |  |  |  |
| bot | varchar(50) | NULL | NO |  |  |  |
| flag | varchar(50) | NULL | NO |  |  |  |
| content | json | NULL | NO |  |  |  |
| priority | int | NULL | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| expire_at | datetime | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## jobs 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| workflow_id | bigint | 0 | NO | MUL | DEFAULT_GENERATED |  |
| dag_id | bigint | 0 | NO |  | DEFAULT_GENERATED |  |
| trigger_id | bigint | 0 | NO |  | DEFAULT_GENERATED |  |
| script_version | smallint | 0 | NO |  | DEFAULT_GENERATED |  |
| state | tinyint | NULL | NO | MUL |  |  |
| started_at | datetime | NULL | YES |  |  |  |
| ended_at | datetime | NULL | YES |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## key_result_values 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| key_result_id | bigint | NULL | YES | MUL |  |  |
| value | int | NULL | NO |  |  |  |
| memo | varchar(1000) |  | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## key_results 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| objective_id | bigint | 0 | NO |  | DEFAULT_GENERATED |  |
| sequence | int | NULL | NO |  |  |  |
| title | varchar(100) | NULL | NO |  |  |  |
| memo | varchar(1000) | NULL | NO |  |  |  |
| initial_value | int | NULL | NO |  |  |  |
| target_value | int | NULL | NO |  |  |  |
| current_value | int | NULL | NO |  |  |  |
| value_mode | varchar(20) |  | NO |  |  |  |
| tag | varchar(100) | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## messages 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| flag | char(36) | NULL | NO | UNI |  |  |
| platform_id | bigint | 0 | NO | MUL | DEFAULT_GENERATED |  |
| platform_msg_id | varchar(50) |  | NO |  |  |  |
| topic | char(36) | NULL | NO | MUL |  |  |
| content | json | NULL | YES |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| deleted_at | datetime | NULL | YES |  |  |  |


## oauth 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| name | varchar(100) | NULL | NO |  |  |  |
| type | varchar(50) | NULL | NO |  |  |  |
| token | varchar(256) | NULL | NO |  |  |  |
| extra | json | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## objectives 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| sequence | int | NULL | NO |  |  |  |
| progress | tinyint | 0 | NO |  |  |  |
| title | varchar(100) | NULL | NO |  |  |  |
| memo | varchar(1000) | NULL | NO |  |  |  |
| motive | varchar(1000) | NULL | NO |  |  |  |
| feasibility | varchar(1000) | NULL | NO |  |  |  |
| is_plan | tinyint | 0 | NO |  |  |  |
| plan_start | date | NULL | NO |  |  |  |
| plan_end | date | NULL | NO |  |  |  |
| total_value | int | NULL | NO |  |  |  |
| current_value | int | NULL | NO |  |  |  |
| tag | varchar(100) | NULL | NO |  |  |  |
| created_data | datetime | NULL | NO |  |  |  |
| updated_date | datetime | NULL | NO |  |  |  |


## pages 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| page_id | varchar(100) | NULL | NO | MUL |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| type | varchar(100) | NULL | NO |  |  |  |
| schema | json | NULL | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## parameter 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| flag | char(25) | NULL | NO | UNI |  |  |
| params | json | NULL | YES |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| expired_at | datetime | NULL | NO |  |  |  |


## platform_bots 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| platform_id | bigint | 0 | NO | MUL |  |  |
| bot_id | bigint | 0 | NO | MUL |  |  |
| flag | varchar(50) | 0 | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## platform_channels 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| platform_id | bigint | 0 | NO | MUL |  |  |
| channel_id | bigint | 0 | NO | MUL |  |  |
| flag | varchar(50) | 0 | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## platform_users 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| platform_id | bigint | 0 | NO | MUL |  |  |
| user_id | bigint | 0 | NO | MUL |  |  |
| flag | varchar(36) | NULL | NO | MUL |  |  |
| name | varchar(30) | NULL | NO |  |  |  |
| email | varchar(50) | NULL | NO |  |  |  |
| avatar_url | varchar(200) | NULL | NO |  |  |  |
| is_bot | tinyint(1) | 0 | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## platforms 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| name | varchar(50) | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## review_evaluations 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| review_id | bigint | 0 | NO | MUL | DEFAULT_GENERATED |  |
| question | varchar(255) | NULL | NO |  |  |  |
| reason | varchar(255) | NULL | NO |  |  |  |
| solving | varchar(255) | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## reviews 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| objective_id | bigint | 0 | NO |  | DEFAULT_GENERATED |  |
| type | tinyint | NULL | NO |  |  |  |
| rating | tinyint | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## schema_migrations 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| version | int | NULL | NO | PRI | auto_increment |  |
| dirty | tinyint | 0 | NO |  | DEFAULT_GENERATED |  |


## steps 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| job_id | bigint | 0 | NO | MUL | DEFAULT_GENERATED |  |
| action | json | NULL | NO |  |  |  |
| name | varchar(100) |  | NO |  |  |  |
| describe | varchar(300) |  | NO |  |  |  |
| node_id | varchar(50) |  | NO | MUL |  |  |
| depend | json | NULL | YES |  |  |  |
| input | json | NULL | YES |  |  |  |
| output | json | NULL | YES |  |  |  |
| error | varchar(1000) | NULL | YES |  |  |  |
| state | tinyint | NULL | NO | MUL |  |  |
| started_at | datetime | NULL | YES |  |  |  |
| ended_at | datetime | NULL | YES |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## todos 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| key_result_id | bigint | 0 | NO |  | DEFAULT_GENERATED |  |
| parent_id | bigint | 0 | NO | MUL | DEFAULT_GENERATED |  |
| sequence | int | NULL | NO |  |  |  |
| content | varchar(1000) | NULL | NO |  |  |  |
| category | varchar(100) | NULL | NO |  |  |  |
| remark | varchar(100) | NULL | NO |  |  |  |
| priority | int | NULL | NO |  |  |  |
| is_remind_at_time | tinyint | NULL | NO |  |  |  |
| remind_at | bigint | NULL | NO |  |  |  |
| repeat_method | varchar(100) | NULL | NO |  |  |  |
| repeat_rule | varchar(100) | NULL | NO |  |  |  |
| repeat_end_at | bigint | NULL | NO |  |  |  |
| complete | tinyint | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## topics 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint | NULL | NO | PRI | auto_increment |  |
| flag | char(36) |  | NO | UNI |  |  |
| platform | varchar(20) | NULL | NO | MUL |  |  |
| owner | bigint | 0 | NO | MUL |  |  |
| name | char(25) | NULL | NO |  |  |  |
| type | varchar(50) |  | NO |  |  |  |
| tags | json | NULL | YES |  |  |  |
| state | smallint | 0 | NO |  |  |  |
| touched_at | datetime | NULL | YES |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## urls 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| flag | varchar(100) | NULL | NO | MUL |  |  |
| url | varchar(256) | NULL | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| view_count | int | 0 | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## users 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| flag | char(36) | NULL | NO | UNI |  |  |
| name | varchar(50) | NULL | NO |  |  |  |
| tags | json | NULL | YES |  |  |  |
| state | smallint | 0 | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## workflow 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| flag | char(25) | NULL | NO | MUL |  |  |
| name | varchar(100) | NULL | NO |  |  |  |
| describe | varchar(300) | NULL | NO |  |  |  |
| successful_count | int | 0 | NO |  |  |  |
| failed_count | int | 0 | NO |  |  |  |
| running_count | int | 0 | NO |  |  |  |
| canceled_count | int | 0 | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## workflow_script 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| workflow_id | bigint unsigned | NULL | NO |  |  |  |
| lang | varchar(10) | NULL | NO |  |  |  |
| code | text | NULL | NO |  |  |  |
| version | smallint | 1 | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## workflow_trigger 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| workflow_id | bigint | 0 | NO | MUL | DEFAULT_GENERATED |  |
| type | varchar(20) | NULL | NO |  |  |  |
| rule | json | NULL | YES |  |  |  |
| count | int | 0 | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |



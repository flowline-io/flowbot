## agents 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| hostid | varchar(100) | NULL | NO |  |  |  |
| hostname | varchar(100) | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| last_online_at | datetime | NULL | NO |  |  |  |
| online_duration | int | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## behavior 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| count | int | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| extra | json | NULL | YES |  |  |  |
| flag | varchar(100) | NULL | NO | MUL |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## bots 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| name | varchar(50) | NULL | NO |  |  |  |
| state | tinyint | 0 | NO |  | DEFAULT_GENERATED |  |
| updated_at | datetime | NULL | NO |  |  |  |


## channels 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| flag | varchar(36) | NULL | NO | MUL |  |  |
| id | bigint | NULL | NO | PRI | auto_increment |  |
| name | varchar(50) | NULL | NO |  |  |  |
| state | tinyint | 0 | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## configs 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| key | varchar(100) | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| value | json | NULL | NO |  |  |  |


## counter_records 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| counter_id | bigint unsigned | 0 | NO | PRI | DEFAULT_GENERATED |  |
| created_at | datetime | NULL | NO |  |  |  |
| digit | int | NULL | NO |  |  |  |


## counters 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| digit | bigint | NULL | NO |  |  |  |
| flag | varchar(100) | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| status | int | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## cycles 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| end_date | date | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| objectives | json | NULL | NO |  |  |  |
| start_date | date | NULL | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## dag 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| edges | json | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| nodes | json | NULL | NO |  |  |  |
| script_id | bigint | NULL | NO |  |  |  |
| script_version | smallint | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| workflow_id | bigint | 0 | NO | MUL |  |  |


## data 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| key | varchar(100) | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| value | json | NULL | NO |  |  |  |


## fileuploads 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| fid | char(36) | NULL | NO | MUL |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| location | varchar(2048) | NULL | NO |  |  |  |
| mimetype | varchar(255) | NULL | NO |  |  |  |
| name | varchar(255) | NULL | NO |  |  |  |
| size | bigint | NULL | NO |  |  |  |
| state | int | NULL | NO | MUL |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## form 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| extra | json | NULL | YES |  |  |  |
| form_id | varchar(100) | NULL | NO | MUL |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| schema | json | NULL | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| values | json | NULL | YES |  |  |  |


## instruct 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| bot | varchar(50) | NULL | NO |  |  |  |
| content | json | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| expire_at | datetime | NULL | NO |  |  |  |
| flag | varchar(50) | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| no | char(25) | NULL | NO | MUL |  |  |
| object | varchar(20) | NULL | NO |  |  |  |
| priority | int | NULL | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## jobs 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| dag_id | bigint | 0 | NO |  |  |  |
| ended_at | datetime | NULL | YES |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| script_version | smallint | 0 | NO |  |  |  |
| started_at | datetime | NULL | YES |  |  |  |
| state | tinyint | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| trigger_id | bigint | 0 | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| workflow_id | bigint | 0 | NO | MUL |  |  |


## key_result_values 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| key_result_id | bigint | NULL | YES | MUL |  |  |
| memo | varchar(1000) |  | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| value | int | NULL | NO |  |  |  |


## key_results 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| current_value | int | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| initial_value | int | NULL | NO |  |  |  |
| memo | varchar(1000) | NULL | NO |  |  |  |
| objective_id | bigint | 0 | NO |  | DEFAULT_GENERATED |  |
| sequence | int | NULL | NO |  |  |  |
| tag | varchar(100) | NULL | NO |  |  |  |
| target_value | int | NULL | NO |  |  |  |
| title | varchar(100) | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| value_mode | varchar(20) |  | NO |  |  |  |


## messages 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| content | json | NULL | YES |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| deleted_at | datetime | NULL | YES |  |  |  |
| flag | char(36) | NULL | NO | UNI |  |  |
| id | bigint | NULL | NO | PRI | auto_increment |  |
| platform_id | bigint | 0 | NO | MUL | DEFAULT_GENERATED |  |
| platform_msg_id | varchar(50) |  | NO |  |  |  |
| role | varchar(20) | user | NO |  |  |  |
| session | char(36) | NULL | NO | MUL |  |  |
| state | tinyint | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## oauth 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| extra | json | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| name | varchar(100) | NULL | NO |  |  |  |
| token | varchar(256) | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| type | varchar(50) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## objectives 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_data | datetime | NULL | NO |  |  |  |
| current_value | int | NULL | NO |  |  |  |
| feasibility | varchar(1000) | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| is_plan | tinyint | 0 | NO |  |  |  |
| memo | varchar(1000) | NULL | NO |  |  |  |
| motive | varchar(1000) | NULL | NO |  |  |  |
| plan_end | date | NULL | NO |  |  |  |
| plan_start | date | NULL | NO |  |  |  |
| progress | tinyint | 0 | NO |  |  |  |
| sequence | int | NULL | NO |  |  |  |
| tag | varchar(100) | NULL | NO |  |  |  |
| title | varchar(100) | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| total_value | int | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_date | datetime | NULL | NO |  |  |  |


## pages 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| page_id | varchar(100) | NULL | NO | MUL |  |  |
| schema | json | NULL | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| type | varchar(100) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## parameter 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| expired_at | datetime | NULL | NO |  |  |  |
| flag | char(25) | NULL | NO | UNI |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| params | json | NULL | YES |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## platform_bots 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| bot_id | bigint | 0 | NO | MUL |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| flag | varchar(50) | 0 | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| platform_id | bigint | 0 | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## platform_channel_users 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| channel_flag | varchar(50) | NULL | NO | MUL |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| platform_id | bigint | 0 | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| user_flag | varchar(50) | NULL | NO | MUL |  |  |


## platform_channels 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| channel_id | bigint | 0 | NO | MUL |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| flag | varchar(50) | 0 | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| platform_id | bigint | 0 | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## platform_users 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| avatar_url | varchar(200) | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| email | varchar(50) | NULL | NO |  |  |  |
| flag | varchar(36) | NULL | NO | MUL |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| is_bot | tinyint(1) | 0 | NO |  |  |  |
| name | varchar(30) | NULL | NO |  |  |  |
| platform_id | bigint | 0 | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| user_id | bigint | 0 | NO | MUL |  |  |


## platforms 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint | NULL | NO | PRI | auto_increment |  |
| name | varchar(50) | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## review_evaluations 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| question | varchar(255) | NULL | NO |  |  |  |
| reason | varchar(255) | NULL | NO |  |  |  |
| review_id | bigint | 0 | NO | MUL | DEFAULT_GENERATED |  |
| solving | varchar(255) | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## reviews 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| objective_id | bigint | 0 | NO |  | DEFAULT_GENERATED |  |
| rating | tinyint | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| type | tinyint | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## schema_migrations 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| dirty | tinyint(1) | NULL | NO |  |  |  |
| version | bigint | NULL | NO | PRI |  |  |


## steps 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| action | json | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| depend | json | NULL | YES |  |  |  |
| describe | varchar(300) |  | NO |  |  |  |
| ended_at | datetime | NULL | YES |  |  |  |
| error | varchar(1000) | NULL | YES |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| input | json | NULL | YES |  |  |  |
| job_id | bigint | 0 | NO | MUL |  |  |
| name | varchar(100) |  | NO |  |  |  |
| node_id | varchar(50) |  | NO | MUL |  |  |
| output | json | NULL | YES |  |  |  |
| started_at | datetime | NULL | YES |  |  |  |
| state | tinyint | NULL | NO | MUL |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## todos 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| category | varchar(100) | NULL | NO |  |  |  |
| complete | tinyint | NULL | NO |  |  |  |
| content | varchar(1000) | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| is_remind_at_time | tinyint | NULL | NO |  |  |  |
| key_result_id | bigint | 0 | NO |  | DEFAULT_GENERATED |  |
| parent_id | bigint | 0 | NO | MUL | DEFAULT_GENERATED |  |
| priority | int | NULL | NO |  |  |  |
| remark | varchar(100) | NULL | NO |  |  |  |
| remind_at | bigint | NULL | NO |  |  |  |
| repeat_end_at | bigint | NULL | NO |  |  |  |
| repeat_method | varchar(100) | NULL | NO |  |  |  |
| repeat_rule | varchar(100) | NULL | NO |  |  |  |
| sequence | int | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## topics 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| flag | char(36) |  | NO | UNI |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| name | char(25) | NULL | NO |  |  |  |
| owner | bigint | 0 | NO | MUL |  |  |
| platform | varchar(20) | NULL | NO | MUL |  |  |
| state | smallint | 0 | NO |  |  |  |
| tags | json | NULL | YES |  |  |  |
| touched_at | datetime | NULL | YES |  |  |  |
| type | varchar(50) |  | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## urls 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| flag | varchar(100) | NULL | NO | MUL |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| state | tinyint | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| url | varchar(256) | NULL | NO |  |  |  |
| view_count | int | 0 | NO |  |  |  |


## users 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| flag | char(36) | NULL | NO | UNI |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| name | varchar(50) | NULL | NO |  |  |  |
| state | smallint | 0 | NO |  |  |  |
| tags | json | NULL | YES |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## webhook 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| created_at | datetime | NULL | NO |  |  |  |
| flag | char(25) | NULL | NO | MUL |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| secret | varchar(64) | NULL | NO | UNI |  |  |
| state | tinyint | NULL | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| trigger_count | int | 0 | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## workflow 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| canceled_count | int | 0 | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| describe | varchar(300) | NULL | NO |  |  |  |
| failed_count | int | 0 | NO |  |  |  |
| flag | char(25) | NULL | NO | MUL |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| name | varchar(100) | NULL | NO |  |  |  |
| running_count | int | 0 | NO |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| successful_count | int | 0 | NO |  |  |  |
| topic | char(36) | NULL | NO |  |  |  |
| uid | char(36) | NULL | NO | MUL |  |  |
| updated_at | datetime | NULL | NO |  |  |  |


## workflow_script 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| code | text | NULL | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| lang | varchar(10) | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| version | smallint | 1 | NO |  |  |  |
| workflow_id | bigint unsigned | NULL | NO |  |  |  |


## workflow_trigger 

| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |
|-------------|------------------|----------------|-------------|------------|----------------|----------------|
| count | int | 0 | NO |  |  |  |
| created_at | datetime | NULL | NO |  |  |  |
| id | bigint unsigned | NULL | NO | PRI | auto_increment |  |
| rule | json | NULL | YES |  |  |  |
| state | tinyint | NULL | NO |  |  |  |
| type | varchar(20) | NULL | NO |  |  |  |
| updated_at | datetime | NULL | NO |  |  |  |
| workflow_id | bigint | 0 | NO | MUL |  |  |



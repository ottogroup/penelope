## Penelope - GCP Backup Solution 

### About
Penelope is a tool, which allows you to back up data stored in GCP automatically. You can create backups from BigQuery datasets and tables as well as from Cloud Storage buckets within Google Cloud Storage. For authentication against GCP services Penelope uses Google service accounts and for identification it assumes that you bring in your own IDP. 

Penelope consists of three main components: 
* A Docker image for a server written in GO providing an API with different methods to create, start, etc. backups
* A web frontend allowing users to easily create and manage backup jobs 
* A PostgreSQL database storing different pieces of information about backup jobs

**Bellow:** Screenshot from Penelope using the backup form to create a backups

![Backup Form](/resources/screenshots/backup_form_screentshot.png?raw=true)

### Getting Started

This repository provides a starter kit to set up Penelope on your own. Penelope uses providers for different purposes, 
for example penelope needs a credential to connect with the configured database (see environment variables). You can 
use penelopes basic secret provider, which uses a specific environment variable to provide a secret credential, or 
you can define a more advance provider, which for example fetches the credentials during runtime. Penelope need four
specific providers:

* `SecretProvider` - containing the method *GetSecret*, which provides the database password for given user.
* `SinkGCPProjectProvider` - containing the method *GetSinkGCPProjectID*, which provides for a given GCP project id a specific cloud storage backup sink.
* `TargetPrincipalForProjectProvider` - contains the method *GetTargetPrincipalForProject*, which provides a target service account to be impersonated for a given project.
* `PrincipalProvider` - contains the method *GetPrincipalForEmail*, which provides the users principal (containing the user and role bindings) for a given email address.

#### Database migrations

Penelope uses a PostgreSQL database to store the backup state. You can find the migrations under the folder `resources/migrations/`. 
You can use [Flyway](https://flywaydb.org/) to run the migrations against your own pg database.

#### Configuration

Penelope uses environment variables for customization. Therefore, you can configure penelope to a certain degree
by setting specific environment variables (e.g. configure database connection). There are optional and required
settings. If you not provide required settings, penelope will not run.

| Name | Required | Description |
| ---- | ---- | ---- |
| `GCP_PROJECT_ID` | required | Set the GCP project. |
| `APP_JWT_AUDIENCE` | required | Set the expected audience value of the jwt token. |
| `COMPANY_DOMAINS` | required | Set the company domains for validating user email. Value can be a comma separated list. |
| `DEFAULT_BUCKET_STORAGE_CLASS` | required | Set the default storage class for backup sinks. |
| `POSTGRES_SOCKET` | required | Set socket address to PostgreSQL server.  |
| `POSTGRES_HOST` | required | Set host address to PostgreSQL server. If PostgreSQL socket is specified, setting this is optional. |
| `POSTGRES_PORT` | required | Set port of PostgreSQL server default to `5432`. |
| `POSTGRES_DB` | required | Set name of PostgreSQL database. |
| `POSTGRES_USER` | required | Set username to connect with PostgreSQL database. |
| `POSTGRES_PASSWORD` | required | Set password for user to connect with PostgreSQL database. |
| `TOKEN_HEADER_KEY` | required | Set the key for token header. |
| `PENELOPE_PORT` | optional | Set port for localhost when running penelope local. |
| `PENELOPE_TRACING` | optional | Set `true` to export tracing metrics to Stackdriver. Default is `true`. |
| `PENELOPE_TRACING_METRICS_PREFIX` | optional | Set prefix for tracing metrics when activated. Default is `penelope-server`. |
| `PENELOPE_USE_DEFAULT_HTTP_CLIENT` | optional | Switch to use default http request for testing by setting `true`. Default is `false`. |
| `CORS_ALLOWED_METHODS` | optional | Set the allowed methods for CORS with a comma separated list. For example, `POST, PATCH, GET` |
| `CORS_ALLOWED_ORIGIN` | optional | Set the allowed origins for defined cors methods. |
| `CORS_ALLOWED_HEADERS` | optional | Set the allowed request headers.  |
| `TASKS_VALIDATION_HTTP_HEADER_NAME` | optional | Adds request validation to tasks triggers. Specifies the expected request head for validation. |
| `TASKS_VALIDATION_HTTP_HEADER_VALUE` | optional | Expected value for request validation.  |
| `TASKS_VALIDATION_ALLOWED_IP_ADDRESSES` | optional | Adds ip address validation to tasks triggers. Multiple comma separated ip addresses can be specified. |

### Deploy Basic Setup

This step-by-step guide will walk you through how to set up Penelope in your own Google App Engine instance. Let us start with the database migration.

#### 1. Step: Migration with Flyway

In the following we show, how you can use Flyway for migration. However, feel free to use any other tool which 
fits best for your use case. The migration files are in the folder `resource/migrations` as already mentioned above.

```shell script
flyway migrate -url=jdbc:postgresql://<HOST>:<PORT>/<DB> -user=<USER> -password=<PW> -locations=filesystem:./resources/migrations
```

Because we are going to deploy Penelope to App Engine, it maybe useful to take 
CloudSQL into consideration. You can use Cloud SQL Proxy to connect with your instance via a
secure connection. In order to find more about how to set up a connection using the proxy client see the
[About the Cloud SQL Proxy] (https://cloud.google.com/sql/docs/mysql/sql-proxy) documentation. 

#### 2. Step: Configure App Engine

You are going to need a `app.yaml` (.yml with an a) file to deploy and configure your App Engine service.
In this file you specify the url handlers and can set the already mentioned environment variables to configure Penelope 
as well.

### API 

The provided API can handle both POST, PATCH and GET requests. They are used for different methods, though. 
When creating new backups you have to use POST and include all backup specifications. Depending on what action you want 
to carry out, you can make use of the following API request types:

#### Users

##### Get user

Returns all project role bindings of the currently logged in user.

```shell script
POST /api/users/me
```

#### Backups

##### Create backup

The *create* request allows you to create a backup, either a one shot or repeating backup sequences.

```shell script
POST /api/backups
```

| Name | Required | Description |
|---|---|---|
| type | required | The type of backup which should be created `BigQuery` or `CloudStorage`. |
| strategy | required | The strategy for the backup which can be `Snapshot` or `Mirror`. |
| project | required | The GCP project for which a backup should be created. | 
| target | required | Options for target backup. | 
| snapshot_options | optional | Options for a `Snapshot` backup. | 
| mirror_options | optional | Options for a `Mirror` backup. | 
| bigquery_options | optional | Specified options when type of backup is `BigQuery`. | 
| gcs_options | optional |  Specified options when type of backup is `CloudStorage`. | 

##### List backup

The *GET* request allows you to get all backups from one GCP project.

```shell script
GET /api/backups
```

| Name | Required | Description |
|---|---|---|
| project | required | The GCP project of the backups. |

##### Get single backup

Returns a single backup, specified by the id parameter.

```shell script
GET /api/backups/:id
```

| Name | Required | Description |
|---|---|---|
| id | required | The ID of the desired backup. |

##### Update single backup

Updates a single backup, specified by the id parameter.

```shell script
PATCH /api/backups/:id
```

| Name | Required | Description |
|---|---|---|
| id | required | The ID of the desired backup. |

##### Calculates cost of single backup

Calculates the cost of a single backup, specified by the parameters.

```shell script
POST /api/backups/calculate
```

| Name | Required | Description |
|---|---|---|
| type | required | The type of backup which should be created `BigQuery` or `CloudStorage`. |
| strategy | required | The strategy for the backup which can be `Snapshot` or `Mirror`. |
| project | required | The GCP project for which a backup should be created. | 
| target | required | Options for target backup. | 
| snapshot_options | optional | Options for a `Snapshot` backup. | 
| mirror_options | optional | Options for a `Mirror` backup. | 
| bigquery_options | optional | Specified options when type of backup is `BigQuery`. | 
| gcs_options | optional |  Specified options when type of backup is `CloudStorage`. | 

#### Buckets & Datasets

##### List cloud storage buckets

Lists all google cloud storage buckets from a GCP project, specified by the id parameter.

```shell script
GET /api/buckets/:id
```

| Name | Required | Description |
|---|---|---|
| id | required | The ID of the desired GCP project. |

##### List big query datasets

Lists all big query datasets from a GCP project, specified by the id parameter.

```shell script
GET /api/datasets/:id
```

| Name | Required | Description |
|---|---|---|
| id | required | The ID of the desired GCP project. |

#### Restore

##### List restore actions

Lists restore actions for a backup. An addition, all previous jobs including the specified job can be chosen.

```shell script
GET /api/restore/:id
```

| Name | Required | Description |
|---|---|---|
| jobIDForTimestamp | optional | When specified, it returns the restore action for the specified and all previous jobs. |

#### Tasks

These are special api endpoint for triggering backup tasks. You may want to set up cron jobs
to trigger these tasks regularly and automatically.

##### Prepare all backup jobs

```shell script
GET /api/tasks/prepare_backup_jobs
```

##### Run new jobs

```shell script
GET /api/tasks/run_new_jobs
```

##### Check all job status 

```shell script
GET /api/tasks/check_jobs_status
```

##### Check stuck jobs 

```shell script
GET /api/tasks/check_jobs_stuck
```

##### Clean up expired backup sinks

```shell script
GET /api/tasks/cleanup_expired_sinks
```

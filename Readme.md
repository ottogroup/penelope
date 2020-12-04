## Penelope - GCP Backup Solution 

### About
Penelope is a tool, which allows you to back up data stored in GCP automatically. You can create backups from BigQuery datasets and tables as well as from Cloud Storage buckets within Google Cloud Storage. For authentication against GCP services Penelope uses Google service accounts and for identification it assumes that you bring in your own IDP. 

Penelope consists of three main components: 
* A Docker image for a server written in GO providing an API with different methods to create, start, etc. backups
* A web frontend allowing users to easily create and manage backup jobs 
* A PostgreSQL database storing different pieces of information about backup jobs

**Bellow:** Screenshot from Penelope using the form to create a new backup

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
| `DEFAULT_PROVIDER_BUCKET` | required | Set the bucket for all providers |
| `DEFAULT_PROVIDER_SINK_FOR_PROJECT_PATH` | required | Set the file path for backup sink for project provider. |
| `DEFAULT_PROVIDER_PRINCIPAL_FOR_USER_PATH` | required | Set the file path for user principal provider. |
| `DEFAULT_PROVIDER_IMPERSONATE_GOOGLE_SERVICE_ACCOUNT` | required | Set default impersonated google service account. |
| `STATIC_FILES_PATH` | required | If Penelope runs locally, set the static files path. |
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
| `PENELOPE_PORT` | optional | Set port for localhost when running Penelope local. |
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

In the following you will learn, how you can use Flyway for migration. However, feel free to use any other tool which 
fits best for your use case. The migration files are in the folder `resource/migrations` as already mentioned above.

```shell script
flyway migrate -url=jdbc:postgresql://<HOST>:<PORT>/<DB> -user=<USER> -password=<PW> -locations=filesystem:./resources/migrations
```

Because we are going to deploy Penelope to App Engine, it maybe useful to take 
CloudSQL into consideration. You can use Cloud SQL Proxy to connect with your instance via a
secure connection. In order to find out more about the proxy client see the
[About the Cloud SQL Proxy](https://cloud.google.com/sql/docs/mysql/sql-proxy) documentation. 

#### 2. Step: Configuration of App Engine

You are going to need a `app.yaml` file to deploy and configure your App Engine service.
In this file you specify the go runtime version, url handlers and can set all environment variables to configure Penelope 
as well. This repository provides a configuration template for your own App Engine. Replace the brackets and feel 
free to change the values, but be carefully with the handlers.

```yaml
# app.yaml
runtime: go114
service: default
handlers:
  -   url: /
      static_files: static/ui/index.html
      upload: static/ui/index.html
  # ...

env_variables:
  GCP_PROJECT_ID: <GCP_PROJECT_ID>
  PENELOPE_PORT: <PENELOPE_PORT>
  POSTGRES_SOCKET: /cloudsql/<GCP_PROJECT>:<REGION>:<DB_INSTANCE>/.s.PGSQL.5432
  POSTGRES_USER: <POSTGRES_USER>
  POSTGRES_DB: <POSTGRES_DB>
  POSTGRES_PASSWORD: <POSTGRES_PASSWORD>
  # ...
```

#### 4. Step: Penelope Deployment

Now that you have specified the configuration for Penelope, you are able to deploy the local application and 
configuration settings by using Cloud SDK. For more details on how to install or manage your GCP resources and 
applications see [Google Cloud SDK Documentation](https://cloud.google.com/sdk/docs/quickstart). Since we are going to
deploy the application to app engine, we will use `gcloud app deploy` for deployment.

```shell script
gcloud app deploy app.yaml
```

#### 3. Step: Configuration of Cron-Jobs

Congratulations. If you configured your application correctly, you successfully deployed Penelope to
App Engine. But you're not done yet. There are still tasks, which need to be triggered. These Penelope tasks are responsible for 
making backups, cleanups of expired sinks and so on. This repository provides a basic cron job configuration as well for all
tasks. There are no changes required, but feel free to change the scheduling.

```yaml
# cron.yaml
cron:
    -   description: "prepare backup jobs"
        url: /api/tasks/prepare_backup_jobs
        schedule: every 60 minutes from 00:00 to 23:00
    -   description: "schedule new jobs"
        url: /api/tasks/run_new_jobs
        schedule: every 10 minutes from 00:05 to 23:55
    # ...
``` 

#### 4. Step: Cron-Jobs Scheduling

Deploying the `cron.yaml` configuration file to App Engine is straight forward. You just need to run the following command.

```shell script
gcloud app deploy cron.yaml
```

### Providers

This section is specifically talking about the special Penelope providers. As mentioned before, there are four 
providers which provide Penelope with information like where to store the backup, which role bindings has the user and so on. 
This repository contains default providers. However, you are able to implement your own provider. In the following, 
you will find out how each default provider works and how you can implement your own provider. To use your own Penelope 
defined providers use `AppStartArguments` and pass it to the run function of the `cmd` package.

```go
package main

import (
    "github.com/ottogroup/penelope/cmd"
)


func main() {
    // Create all your providers here ...

    appStartArguments := app.AppStartArguments{
        SecretProvider:                    secretProvider,
        SinkGCPProjectProvider:            sinkGCPProjectProvider,
        PrincipalProvider:                 principalProvider,
        TargetPrincipalForProjectProvider: targetPrincipalForProjectProvider,
    }

    cmd.Run(appStartArguments)
}
```

#### The Secret Provider 

Let's have a look at the first provider. The secret provider, specified by the `SecretProvider` interface, provides Penelope with the database
password. This provider defines only one method. It receives a `context.Context` and `string` argument and returns
a `string` and `error` type. You can probably guess the meaning of each argument. However, we will go through 
each parameter to be clear. The first argument expected is a context, which is created for each (http) request. This is 
golang specific. If you want to find out more about the Context type, you can read the [Package Context](https://golang.org/pkg/context/)
documentation. The next argument contains the database user name. All you have to do is to return the 
password for this user. If you are not able to return the database password, you can return an error value.


```go
package secret

import "context"

type SecretProvider interface {
  GetSecret(ctxIn context.Context, user string) (string, error)
}
```

##### Default

The default provider is actually pretty straight forward. It basically doesn't care about the user argument. It just returns the 
value you have specified in the `POSTGRES_PASSWORD` environment variable. You think this is bad? Then feel free to 
define your own implementation.  

#### Backup Provider 

The tasks of the backup sink provider is to provide Penelope with a GCP project where the backup should be stored.
This provider is defined by the `SinkGCPProjectProvider` interface. The first argument is the same for all provider methods, 
which is again context. The next argument is the source GCP project id. It is the project of the source data, which should
be backup on a target project. The target project is actually defined by the return value.

```go
package provider

import "context"

type SinkGCPProjectProvider interface {
    GetSinkGCPProjectID(ctxIn context.Context, sourceGCPProjectID string) (string, error)
}
```

##### Default

The default provide is a bit more complex this time. You not only have to define the environment variables 
`DEFAULT_PROVIDER_BUCKET` and `DEFAULT_PROVIDER_SINK_FOR_PROJECT_PATH`, you also have to store a `.yaml` file 
in the bucket. The content can look like this.

```yaml
- project: project-one
  backup: project-one-backup
- project: project-two
  backup: project-two-backup
```

For each project you define a backup project (actually not that complex, huh?). But what happens, if a source 
project is not listed in the file? Then the default implementation returns an error. You think there are better solutions.
Maybe you would like to create a backup projects on-the-fly or just use the source project as the target project. Then 
feel free to implement your own `SinkGCPProjectProvider`.

#### Target Principal Provider

This provider can be more difficult to comprehend than the previous providers. Behind the scenes, Penelope uses 
impersonation to create all the backup sinks and so on in GCP. And what does it impersonate to do all these tasks? 
[Service accounts](https://cloud.google.com/iam/docs/understanding-service-accounts), which are special google account
to represent non-human user like applications. To determine which service account should be impersonated by Penelope, 
the `TargetPrincipalForProjectProvider` interface is required. It returns the service account for a target project. 

```go
package impersonate

import "context"

type TargetPrincipalForProjectProvider interface {
    GetTargetPrincipalForProject(ctxIn context.Context, projectID string) (string, error)
}
```

##### Default

The default is again pretty straight forward. You only have to define one single google service account, which should
be impersonated. This is done by setting the `DEFAULT_PROVIDER_IMPERSONATE_GOOGLE_SERVICE_ACCOUNT` environment variable.

#### Principal Provider

The final provider provides the users principal. What is meant by user principal? Lets find out
by looking at the `Principal` data type. The Principal consist of the users email (which is a string) and a list 
of role bindings. The role bindings, in turn, consist of project id and users role for this project. A user can have
one of three roles for a project `None`, `Viewer` or `Owner`.  Let's take a step back and look what `PrincipalProvider` 
actually does. It returns the users roles for each project. Why is this important? Because a user can only do a backup,
if he is the `Owner` of a project. Without the user has no right to edit any data of the project. The `PrincipalProvider`
interface consist of one method, which receives an email address and returns the users principal data.


```go
package provider

import (
	"context"
    "github.com/ottogroup/penelope/pkg/http/auth/model"
)

type PrincipalProvider interface {
    GetPrincipalForEmail(ctxIn context.Context, email string) (*model.Principal, error)
}
```

The data type `Principal` is shown in following source code, which contains additionally all relevant information. 
You can see it consist of a `User` and a list of `ProjectRoleBinding`s. Furthermore, you can see that `User` only 
consists of the email address. The `ProjectRoleBinding` contains the role for each project.

```go
package model

type Role string

type Principal struct {
    User         User
    RoleBindings []ProjectRoleBinding 
}

type User struct {
    Email string
}

var (
    None Role = "none"
    Viewer Role = "viewer"
    Owner Role = "owner"
)

type ProjectRoleBinding struct {
    Role    Role
    Project string
}
```

##### Default

Now let's have a look at the default implementation. The default is very similar to the `SinkGCPProjectProvider`. It 
also needs the path to a `.yaml` file. Therefore `DEFAULT_PROVIDER_PRINCIPAL_FOR_USER_PATH` needs to be set.
The content can look like this. 

```yaml
- user:
    email: 'first-user@example.de'
  role_bindings:
    - role: owner
      project: 'project-one'
    - role: viewer
      project: 'project-two'
- user:
    email: 'second-user@example.de'
  role_bindings:
    - role: viewer
      project: 'project-one'
    - role: viewer
      project: 'project-two'
```

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

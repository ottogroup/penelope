CREATE TABLE IF NOT EXISTS groups
(
  id                      SERIAL PRIMARY KEY,
  email                   TEXT CHECK (email IS NOT NULL),

  audit_created_timestamp timestamp DEFAULT NOW(),
  audit_updated_timestamp timestamp,
  audit_deleted_timestamp timestamp,
  UNIQUE (id),
  UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS users
(
  id                      SERIAL PRIMARY KEY,
  email                   TEXT CHECK (email IS NOT NULL),
  active                  bool CHECK (active IS NOT NULL),
  authtoken               TEXT,

  audit_created_timestamp timestamp DEFAULT NOW(),
  audit_updated_timestamp timestamp,
  audit_deleted_timestamp timestamp,
  UNIQUE (id)
);


CREATE TABLE IF NOT EXISTS users_groups
(
  user_id  int REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
  group_id int REFERENCES groups (id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT user_group_pkey PRIMARY KEY (user_id, group_id)
);

package migration

// Migration0001 is the initial migration.
var Migration0001 = Migration{
	Version: 1,
	Name:    "create tables",
	SQL: `
CREATE TABLE IF NOT EXISTS charm_user(
  id INTEGER NOT NULL PRIMARY KEY,
  charm_id uuid UNIQUE NOT NULL,
  name varchar(50) UNIQUE,
  email varchar(254),
  bio varchar(1000),
    created_at timestamp default current_timestamp
);

CREATE TABLE IF NOT EXISTS public_key(
  id INTEGER NOT NULL PRIMARY KEY,
  user_id integer NOT NULL,
  public_key varchar(2048) NOT NULL,
    created_at timestamp default current_timestamp,
  UNIQUE (user_id, public_key),
  CONSTRAINT user_id_fk
  FOREIGN KEY (user_id)
  REFERENCES charm_user (id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS encrypt_key(
  id INTEGER NOT NULL PRIMARY KEY,
  public_key_id integer NOT NULL,
  global_id uuid NOT NULL,
    created_at timestamp default current_timestamp,
  encrypted_key varchar(2048) NOT NULL,
  CONSTRAINT public_key_id_fk
  FOREIGN KEY (public_key_id)
  REFERENCES public_key (id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS named_seq(
  id INTEGER NOT NULL PRIMARY KEY,
  user_id integer NOT NULL,
  seq integer NOT NULL DEFAULT 0,
  name varchar(1024) NOT NULL,
  UNIQUE (user_id, name),
  CONSTRAINT user_id_fk
  FOREIGN KEY (user_id)
  REFERENCES charm_user (id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS news(
  id INTEGER NOT NULL PRIMARY KEY,
  subject text,
  body text,
  created_at timestamp default current_timestamp
);

CREATE TABLE IF NOT EXISTS news_tag(
  id INTEGER NOT NULL PRIMARY KEY,
  tag varchar(250),
  news_id integer NOT NULL,
  CONSTRAINT news_id_fk
  FOREIGN KEY (news_id)
  REFERENCES news (id)
  ON DELETE CASCADE
  ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS token(
  id INTEGER NOT NULL PRIMARY KEY,
  pin text UNIQUE NOT NULL,
  created_at timestamp default current_timestamp
);
`,
}

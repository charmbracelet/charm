package migration

// Migration0002 is the initial inclusion of foreign keys.
var Migration0002 = Migration{
	Version: 2,
	Name:    "foreign keys",
	SQL: `
PRAGMA foreign_keys=off;

/* public_key */
ALTER TABLE public_key RENAME TO _public_key;

CREATE TABLE public_key(
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

INSERT INTO public_key SELECT * FROM _public_key;
/* public_key */

/* encrypt_key */
ALTER TABLE encrypt_key RENAME TO _encrypt_key;

CREATE TABLE encrypt_key(
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

INSERT INTO encrypt_key SELECT * FROM _encrypt_key;
/* encrypt_key */

/* named_seq */
ALTER TABLE named_seq RENAME TO _named_seq;

CREATE TABLE named_seq(
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

INSERT INTO named_seq SELECT * FROM _named_seq;
/* named_seq */

/* news_tag */
ALTER TABLE news_tag RENAME TO _news_tag;

CREATE TABLE news_tag(
	id INTEGER NOT NULL PRIMARY KEY,
	tag varchar(250),
	news_id integer NOT NULL,
	CONSTRAINT news_id_fk
		FOREIGN KEY (news_id)
		REFERENCES news (id)
		ON DELETE CASCADE
		ON UPDATE CASCADE
);

INSERT INTO news_tag SELECT * FROM _news_tag;
/* news_tag */

DROP TABLE _public_key;
DROP TABLE _encrypt_key;
DROP TABLE _named_seq;
DROP TABLE _news_tag;

PRAGMA foreign_keys=on;
`,
}

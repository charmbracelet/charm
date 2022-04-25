package sqlite

const (
	sqlCreateUserTable = `CREATE TABLE IF NOT EXISTS charm_user(
                        id INTEGER NOT NULL PRIMARY KEY,
                        charm_id uuid UNIQUE NOT NULL,
                        name varchar(50) UNIQUE,
                        email varchar(254),
                        bio varchar(1000),
                        created_at timestamp default current_timestamp
                        )`

	sqlCreatePublicKeyTable = `CREATE TABLE IF NOT EXISTS public_key(
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
                            )`

	sqlCreateEncryptKeyTable = `CREATE TABLE IF NOT EXISTS encrypt_key(
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
                            )`

	sqlCreateNamedSeqTable = `CREATE TABLE IF NOT EXISTS named_seq(
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
                            )`

	sqlCreateNewsTable = `CREATE TABLE IF NOT EXISTS news(
                        id INTEGER NOT NULL PRIMARY KEY,
                        subject text,
                        body text,
                        created_at timestamp default current_timestamp
                        )`

	sqlCreateNewsTagTable = `CREATE TABLE IF NOT EXISTS news_tag(
                           id INTEGER NOT NULL PRIMARY KEY,
                           tag varchar(250),
                           news_id integer NOT NULL,
                           CONSTRAINT news_id_fk
                                FOREIGN KEY (news_id)
                                REFERENCES news (id)
                                ON DELETE CASCADE
                                ON UPDATE CASCADE
                           )`

	sqlCreateTokenTable = `CREATE TABLE IF NOT EXISTS token(
                           id INTEGER NOT NULL PRIMARY KEY,
                           pin text UNIQUE NOT NULL,
                           created_at timestamp default current_timestamp
                           )`

	sqlSelectUserWithName         = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE name like ?`
	sqlSelectUserWithCharmID      = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE charm_id = ?`
	sqlSelectUserWithID           = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE id = ?`
	sqlSelectUserPublicKeys       = `SELECT id, public_key, created_at FROM public_key WHERE user_id = ?`
	sqlSelectNumberUserPublicKeys = `SELECT count(*) FROM public_key WHERE user_id = ?`
	sqlSelectPublicKey            = `SELECT id, user_id, public_key FROM public_key WHERE public_key = ?`
	sqlSelectEncryptKey           = `SELECT global_id, encrypted_key, created_at FROM encrypt_key WHERE public_key_id = ? AND global_id = ?`
	sqlSelectEncryptKeys          = `SELECT global_id, encrypted_key, created_at FROM encrypt_key WHERE public_key_id = ? ORDER BY created_at ASC`
	sqlSelectNamedSeq             = `SELECT seq FROM named_seq WHERE user_id = ? AND name = ?`

	sqlInsertUser = `INSERT INTO charm_user (charm_id) VALUES (?)`

	sqlInsertPublicKey = `INSERT INTO public_key (user_id, public_key) VALUES (?, ?)
                        ON CONFLICT (user_id, public_key) DO UPDATE SET
                        user_id = excluded.user_id,
                        public_key = excluded.public_key`
	sqlInsertNews    = `INSERT INTO news (subject, body) VALUES (?,?)`
	sqlInsertNewsTag = `INSERT INTO news_tag (news_id, tag) VALUES (?,?)`

	sqlIncNamedSeq = `INSERT INTO named_seq (user_id, name)
                    VALUES(?,?)
                    ON CONFLICT (user_id, name) DO UPDATE SET
                    user_id = excluded.user_id,
                    name = excluded.name,
                    seq = seq + 1`

	sqlInsertEncryptKey         = `INSERT INTO encrypt_key (encrypted_key, global_id, public_key_id) VALUES (?, ?, ?)`
	sqlInsertEncryptKeyWithDate = `INSERT INTO encrypt_key (encrypted_key, global_id, public_key_id, created_at) VALUES (?, ?, ?, ?)`

	sqlInsertToken = `INSERT INTO token (pin) VALUES (?)`

	sqlUpdateUser            = `UPDATE charm_user SET name = ? WHERE charm_id = ?`
	sqlUpdateMergePublicKeys = `UPDATE public_key SET user_id = ? WHERE user_id = ?`

	sqlDeleteUserPublicKey = `DELETE FROM public_key WHERE user_id = ? AND public_key = ?`
	sqlDeleteUser          = `DELETE FROM charm_user WHERE id = ?`

	sqlDeleteToken = `DELETE FROM token WHERE pin = ?`

	sqlCountUsers     = `SELECT COUNT(*) FROM charm_user`
	sqlCountUserNames = `SELECT COUNT(*) FROM charm_user WHERE name <> ''`

	sqlSelectNews     = `SELECT id, subject, body, created_at FROM news WHERE id = ?`
	sqlSelectNewsList = `SELECT n.id, n.subject, n.created_at FROM news AS n
	                     INNER JOIN news_tag AS t ON t.news_id = n.id
	                     WHERE t.tag = ?
	                     ORDER BY n.created_at desc
	                     LIMIT 50 OFFSET ?`
)

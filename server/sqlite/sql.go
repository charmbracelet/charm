package sqlite

const (
	sqlCreateUserTable = `CREATE TABLE IF NOT EXISTS charm_user(
												id INTEGER NOT NULL PRIMARY KEY,
												charm_id uuid UNIQUE NOT NULL,
												name varchar(50) UNIQUE,
												email varchar(254),
												bio varchar(1000),
												created_at timestamp default current_timestamp)`

	sqlCreatePublicKeyTable = `CREATE TABLE IF NOT EXISTS public_key(
														 id INTEGER NOT NULL PRIMARY KEY,
														 user_id integer REFERENCES charm_user (id) NOT NULL,
														 public_key varchar(2048) NOT NULL,
														 created_at timestamp default current_timestamp,
														 UNIQUE (user_id, public_key))`

	sqlCreateEncryptKeyTable = `CREATE TABLE IF NOT EXISTS encrypt_key(
															id INTEGER NOT NULL PRIMARY KEY,
															public_key_id integer REFERENCES public_key (id) ON DELETE CASCADE NOT NULL,
															global_id uuid NOT NULL,
															encrypted_key varchar(2048) NOT NULL)`

	sqlSelectUserWithName         = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE name like ?`
	sqlSelectUserWithCharmID      = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE charm_id = ?`
	sqlSelectUserWithID           = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE id = ?`
	sqlSelectUserPublicKeys       = `SELECT id, public_key, created_at FROM public_key WHERE user_id = ?`
	sqlSelectNumberUserPublicKeys = `SELECT count(*) FROM public_key WHERE user_id = ?`
	sqlSelectPublicKey            = `SELECT id, user_id, public_key FROM public_key WHERE public_key = ?`
	sqlSelectEncryptKey           = `SELECT global_id, encrypted_key FROM encrypt_key WHERE public_key_id = ? AND global_id = ?`
	sqlSelectEncryptKeys          = `SELECT global_id, encrypted_key FROM encrypt_key WHERE public_key_id = ?`

	sqlInsertUser         = `INSERT INTO charm_user (charm_id) VALUES (?)`
	sqlInsertUserWithName = `INSERT INTO charm_user (charm_id, name) VALUES (?, ?)`
	sqlInsertPublicKey    = `INSERT INTO public_key (user_id, public_key)
													 VALUES (?, ?)
													 ON CONFLICT (user_id, public_key) DO UPDATE SET
													 user_id = excluded.user_id,
													 public_key = excluded.public_key`

	sqlInsertEncryptKey = `INSERT INTO encrypt_key (encrypted_key, global_id, public_key_id) VALUES (?, ?, ?)`

	sqlUpdateUser            = `UPDATE charm_user SET name = ? WHERE charm_id = ?`
	sqlUpdateMergePublicKeys = `UPDATE public_key SET user_id = ? WHERE user_id = ?`

	sqlDeleteUserPublicKey = `DELETE FROM public_key WHERE user_id = ? AND public_key = ?`
	sqlDeleteUser          = `DELETE FROM charm_user WHERE id = ?`

	sqlCountUsers     = `SELECT COUNT(*) FROM charm_user`
	sqlCountUserNames = `SELECT COUNT(*) FROM charm_user WHERE name <> ''`
)

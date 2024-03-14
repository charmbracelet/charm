package sqlite

const (
	sqlCreateVersionTable = `
		CREATE TABLE IF NOT EXISTS version (
			id INTEGER PRIMARY KEY,
			version INTEGER NOT NULL,
			name TEXT NOT NULL,
			completed_at DATETIME,
			error_at DATETIME,
			comment TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(version)
		);`
	sqlDropVersionTable             = `DROP TABLE IF EXISTS version;`
	sqlSelectVersionTableCount      = `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='version';`
	sqlSelectVersionCount           = `SELECT count(*) FROM version;`
	sqlSelectLatestVersion          = `SELECT version, name, completed_at, error_at, comment, created_at, updated_at FROM version ORDER BY version DESC LIMIT 1;`
	sqlSelectIncompleteVersionCount = `SELECT count(*) FROM version WHERE completed_at IS NULL;`
	sqlInsertVersion                = `INSERT INTO version (version, name, comment) VALUES (?, ?, ?);`
	sqlUpdateCompleteVersion        = `UPDATE version SET completed_at = CURRENT_TIMESTAMP WHERE version = ?;`
	sqlUpdateErrorVersion           = `UPDATE version SET error_at = CURRENT_TIMESTAMP, comment = ? WHERE version = ?;`
	sqlSelectUserWithName           = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE name like ?`
	sqlSelectUserWithCharmID        = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE charm_id = ?`
	sqlSelectUserWithID             = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE id = ?`
	sqlSelectUserPublicKeys         = `SELECT id, public_key, created_at FROM public_key WHERE user_id = ?`
	sqlSelectNumberUserPublicKeys   = `SELECT count(*) FROM public_key WHERE user_id = ?`
	sqlSelectPublicKey              = `SELECT id, user_id, public_key FROM public_key WHERE public_key = ?`
	sqlSelectEncryptKey             = `SELECT global_id, encrypted_key, created_at FROM encrypt_key WHERE public_key_id = ? AND global_id = ?`
	sqlSelectEncryptKeys            = `SELECT global_id, encrypted_key, created_at FROM encrypt_key WHERE public_key_id = ? ORDER BY created_at ASC`
	sqlSelectNamedSeq               = `SELECT seq FROM named_seq WHERE user_id = ? AND name = ?`

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
	sqlSelectNewsList = `
		SELECT n.id, n.subject, n.created_at FROM news AS n
		INNER JOIN news_tag AS t ON t.news_id = n.id
		WHERE t.tag = ?
		ORDER BY n.created_at desc
		LIMIT 50 OFFSET ?`
)

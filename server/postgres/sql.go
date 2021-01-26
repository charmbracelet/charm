package postgres

const (
	sqlCreateUserTable = `CREATE TABLE IF NOT EXISTS charm_user(
			      id serial CONSTRAINT user_pk PRIMARY KEY,
			      charm_id uuid UNIQUE NOT NULL,
			      name varchar(50) UNIQUE,
			      email varchar(254),
			      bio varchar(1000),
			      created_at timestamp default current_timestamp)`

	sqlCreatePublicKeyTable = `CREATE TABLE IF NOT EXISTS public_key(
				   id serial CONSTRAINT public_key_pk PRIMARY KEY,
				   user_id integer REFERENCES charm_user (id) NOT NULL,
				   public_key varchar(2048) NOT NULL,
				   created_at timestamp default current_timestamp,
				   UNIQUE (user_id, public_key))`

	sqlCreateEncryptKeyTable = `CREATE TABLE IF NOT EXISTS encrypt_key(
				    id serial CONSTRAINT encrypt_key_pk PRIMARY KEY,
				    public_key_id integer REFERENCES public_key (id) ON DELETE CASCADE NOT NULL,
				    global_id uuid NOT NULL,
				    encrypted_key varchar(2048) NOT NULL)`

	sqlCreateMarkdownTable = `CREATE TABLE IF NOT EXISTS markdown(
				  id serial CONSTRAINT markdown_pk PRIMARY KEY,
		 	          note text,
				  body text,
				  encrypt_key_global_id uuid,
				  created_at timestamp default current_timestamp)`

	sqlCreateStashTable = `CREATE TABLE IF NOT EXISTS stash(
			       user_id integer REFERENCES charm_user (id) ON DELETE CASCADE,
			       markdown_id integer REFERENCES markdown (id) ON DELETE CASCADE,
			       created_at timestamp default current_timestamp,
			       PRIMARY KEY (user_id, markdown_id))`

	sqlCreateNewsTable = `CREATE TABLE IF NOT EXISTS news(
			      user_id integer REFERENCES charm_user (id) ON DELETE CASCADE,
			      markdown_id integer REFERENCES markdown (id) ON DELETE CASCADE,
			      created_at timestamp default current_timestamp,
			      PRIMARY KEY (user_id, markdown_id))`

	sqlSelectUserWithName         = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE name ilike $1`
	sqlSelectUserWithCharmID      = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE charm_id = $1`
	sqlSelectUserWithID           = `SELECT id, charm_id, name, email, bio, created_at FROM charm_user WHERE id = $1`
	sqlSelectUserPublicKeys       = `SELECT id, public_key, created_at FROM public_key WHERE user_id = $1`
	sqlSelectNumberUserPublicKeys = `SELECT count(*) FROM public_key WHERE user_id = $1`
	sqlSelectPublicKey            = `SELECT id, user_id, public_key FROM public_key WHERE public_key = $1`
	sqlSelectEncryptKey           = `SELECT global_id, encrypted_key FROM encrypt_key WHERE public_key_id = $1 AND global_id = $2`
	sqlSelectEncryptKeys          = `SELECT global_id, encrypted_key FROM encrypt_key WHERE public_key_id = $1`

	sqlSelectStashMarkdown = `SELECT md.id, md.body, md.note, md.created_at, md.encrypt_key_global_id FROM markdown AS md
				  LEFT JOIN stash AS s ON md.id = s.markdown_id
				  WHERE s.user_id = $1
				  AND md.id = $2
				  LIMIT 1`

	sqlSelectNewsMarkdown = `SELECT md.id, md.body, md.note, md.created_at, '' FROM markdown AS md
			         LEFT JOIN news AS s ON md.id = s.markdown_id
			         WHERE md.id = $1
			         LIMIT 1`

	sqlSelectStash = `SELECT md.id, md.note, md.created_at, md.encrypt_key_global_id FROM markdown AS md
			  LEFT JOIN stash AS s ON md.id = s.markdown_id
			  WHERE s.user_id = $1
			  ORDER BY md.created_at desc
			  LIMIT 50 OFFSET $2`

	sqlSelectNews = `SELECT md.id, md.note, md.created_at, '' FROM markdown AS md
			 LEFT JOIN news AS s ON md.id = s.markdown_id
			 WHERE s.user_id in (SELECT id FROM charm_user WHERE name in ('toby', 'christian', 'muesli'))
			 ORDER BY md.created_at desc
			 LIMIT 50 OFFSET $1`

	sqlInsertStashMarkdown = `WITH minsert AS (
				  INSERT INTO markdown(note, body, encrypt_key_global_id)
				  VALUES ($2, $3, $4)
				  RETURNING id as markdown_id)
				  INSERT INTO stash (markdown_id, user_id)
				  SELECT markdown_id, $1 FROM minsert
				  RETURNING markdown_id`

	sqlInsertUser         = `INSERT INTO charm_user (charm_id) VALUES ($1)`
	sqlInsertUserWithName = `INSERT INTO charm_user (charm_id, name) VALUES ($1, $2)`
	sqlInsertPublicKey    = `INSERT INTO public_key (user_id, public_key)
				 VALUES ($1, $2)
				 ON CONFLICT (user_id, public_key) DO UPDATE SET
				 user_id = COALESCE($1, excluded.user_id),
				 public_key = COALESCE($2, excluded.public_key)`

	sqlInsertEncryptKey = `INSERT INTO encrypt_key (encrypted_key, global_id, public_key_id)
			       VALUES ($1, $2, $3)
			       RETURNING id`

	sqlInsertNewsMarkdown = `WITH minsert AS (
				 INSERT INTO markdown(note, body)
				 VALUES ($2, $3)
				 RETURNING id as markdown_id)
				 INSERT INTO news (markdown_id, user_id)
				 SELECT markdown_id, $1 FROM minsert`

	sqlUpdateUser            = `UPDATE charm_user SET name = $2 WHERE charm_id = $1`
	sqlUpdateMergePublicKeys = `UPDATE public_key SET user_id = $1 WHERE user_id = $2`
	sqlUpdateMergeStash      = `UPDATE stash SET user_id = $1 WHERE user_id = $2`
	sqlUpdateStashMarkdown   = `UPDATE markdown SET note = $1 WHERE id = $2`

	sqlDeleteUserPublicKey     = `DELETE FROM public_key WHERE user_id = $1 AND public_key = $2`
	sqlDeleteUser              = `DELETE FROM charm_user WHERE id = $1`
	sqlDeleteStashMarkdown     = `DELETE FROM markdown WHERE id in (SELECT markdown_id FROM stash WHERE user_id = $1 AND markdown_id = $2)`
	sqlDeleteUserStashMarkdown = `DELETE FROM markdown WHERE id in (SELECT markdown_id FROM stash WHERE user_id = $1)`

	sqlCountUsers     = `SELECT COUNT(*) FROM charm_user`
	sqlCountUserNames = `SELECT COUNT(*) FROM charm_user WHERE name <> ''`
	sqlCountStashes   = `SELECT COUNT(*) FROM stash`
)

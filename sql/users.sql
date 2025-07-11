CREATE TABLE users.users (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	create_time timestamptz DEFAULT now() NOT NULL,
	update_time timestamptz DEFAULT now() NOT NULL,
	last_visited timestamptz DEFAULT now() NOT NULL,		
	user_agent text NULL,
	email text,
	name text,
	description text,
	CONSTRAINT users_pkey PRIMARY KEY (id)
);

CREATE OR REPLACE FUNCTION users.set_user(user_data json)
RETURNS json AS $$
DECLARE 
    res json;
		l_id uuid;
		l_email text;
		l_user_agent text;
		l_name text;
		l_description text;
BEGIN
		l_id := shared.set_null_if_empty(user_data->>'id')::uuid;
    l_email = shared.set_null_if_empty(user_data->>'email');
    l_user_agent = shared.set_null_if_empty(user_data->>'userAgent');
    l_description = shared.set_null_if_empty(user_data->>'description');
    l_name = shared.set_null_if_empty(user_data->>'name');
    INSERT INTO users.users (
				id,
        email,
        user_agent,
        "name",
				description
    )
    VALUES (
				l_id,
				l_email,
        l_user_agent,
				l_name,
				l_description
    )
    ON CONFLICT (id) DO UPDATE SET
        update_time = now(),
        last_visited = now(),
        email = COALESCE(EXCLUDED.email, users.users.email),
        user_agent = COALESCE(EXCLUDED.user_agent, users.users.user_agent),
        "name" = COALESCE(EXCLUDED.name, users.users.name),
        "description" = COALESCE(EXCLUDED.description, users.users.description)
    RETURNING json_build_object(
        'id', users.users.id,
        'email', users.users.email,
        'name', users.users.name
    ) INTO res;

    RETURN res;
END;
$$ LANGUAGE plpgsql;

select * from users.users;



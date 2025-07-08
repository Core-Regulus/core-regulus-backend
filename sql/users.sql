CREATE TABLE users.users (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	create_time timestamptz DEFAULT now() NOT NULL,
	update_time timestamptz DEFAULT now() NOT NULL,
	last_visited timestamptz NULL,		
	user_agent text NULL,
	CONSTRAINT users_pkey PRIMARY KEY (id)
);

alter table users.users add column email text;
alter table users.users add column name text;
alter table users.users alter column last_visited set default now();
alter table users.users alter column last_visited set not null;


CREATE OR REPLACE FUNCTION users.set_user(user_data json)
RETURNS json AS $$
DECLARE 
    res json;
BEGIN
    INSERT INTO users.users (
				id,
        email,
        user_agent,
        "name"
    )
    VALUES (
				coalesce((user_data->>'id')::uuid, gen_random_uuid()),
        user_data->>'email',
        user_data->>'userAgent',
        user_data->>'name'
    )
    ON CONFLICT (id) DO UPDATE SET
        update_time = now(),
        last_visited = now(),
        email = COALESCE(EXCLUDED.email, users.users.email),
        user_agent = COALESCE(EXCLUDED.user_agent, users.users.user_agent),
        "name" = COALESCE(EXCLUDED.name, users.users.name)
    RETURNING json_build_object(
        'id', users.users.id,
        'email', users.users.email,
        'name', users.users.name
    ) INTO res;

    RETURN res;
END;
$$ LANGUAGE plpgsql;


select users.set_user('{  
	"name": "Test",
	"email": "test@test.com",
	"userAgent": "Test Agent"
}')

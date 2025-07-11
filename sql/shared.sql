create schema shared;

CREATE OR REPLACE FUNCTION shared.is_empty(value text)
RETURNS bool AS $$
DECLARE 
BEGIN
		return (value is null) or (value = '');
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION shared.set_null_if_empty(value text)
RETURNS text AS $$
DECLARE 
BEGIN
	if (shared.is_empty(value)) then
		return null;
	end if;
	return value;
END;
$$ LANGUAGE plpgsql;

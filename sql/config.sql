create schema config;

create table config.config (
	code text primary key,
	value jsonb not null
);

CREATE OR REPLACE FUNCTION config.get()
 RETURNS jsonb
 LANGUAGE plpgsql
AS $function$
declare
	res jsonb;
begin  
	select jsonb_object_agg(code, value) 
	from config.config
	into res;
	return res;
end;
$function$;

insert into config.config (code, value)
values ('google.calendar.id', '"rabinmiller@gmail.com"');
select config.get();








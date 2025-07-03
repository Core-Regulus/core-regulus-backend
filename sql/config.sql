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


CREATE OR REPLACE FUNCTION service.get_days(from_date timestamp with time zone, to_date timestamp with time zone)
 RETURNS TABLE(date date, day_of_week service.day_of_week)
 LANGUAGE plpgsql
AS $function$
declare
	res json;
begin  
	return query
		select d::date,
    			 lower(trim(to_char(d, 'Day')))::service.day_of_week
	  from generate_series(from_date, to_date, interval '1 day') d;
end;
$function$
;


CREATE OR REPLACE FUNCTION service.get_free_slots(date_from timestamp with time zone, date_to timestamp with time zone)
 RETURNS json
 LANGUAGE plpgsql
AS $function$
declare
	res json;
begin  
		with days as (
			select * from service.get_days(date_from, date_to)
		),
		slots as (
			select 
	    	d.date,
		    s.time_start,
	  	  (d.date + s.time_start) as slot_start,
		    (d.date + s.time_start + s.duration) as slot_end
	  	from days d
		  join service.meeting_time_slots s
	  	  on d.day_of_week = s.day_of_week
		)
		select json_agg(json_build_object(
			'date', date,
			'slots', slots
		)) from (
			select date, 
				json_agg(
					json_build_object(
						'timeStart', slot_start, 
						'timeEnd', slot_end
					)
				order by slot_start
			) as slots
		from slots
		where slot_start > now()
		group by date
	)
	into res;
	return coalesce(res, '[]'::json);
end;
$function$
;


select service.get_free_slots('2025-06-01', '2025-06-30');
select * from service.meeting_time_slots; 

select * from config.config;

update service.meeting_time_slots set duration = '45 minutes';




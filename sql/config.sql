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

CREATE OR REPLACE FUNCTION service.get_target_slot(date_from timestamp with time zone)
 RETURNS json
 LANGUAGE plpgsql
AS $function$
declare
	l_res json;
begin  
	select json_build_object(
					'id', id, 
					'dayOfWeek', day_of_week,
					'timeStart', time_start, 
					'duration', extract(epoch from duration)::int
				 )
	from service.meeting_time_slots mts
	into l_res
	where day_of_week = lower(trim(to_char(date_from, 'FMDay')))::service.day_of_week and
				(date_from::time)::interval = mts.time_start and
				date_from > now()
	limit 1;
	return l_res;
end;
$function$;
 

select service.get_target_slot('2025-07-07T09:00:00Z')


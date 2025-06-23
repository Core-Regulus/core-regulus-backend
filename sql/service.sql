create schema service;
CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

create type service.day_of_week as enum ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday');

create table service.meeting_time_slots (
	id uuid primary key not null default gen_random_uuid(),
	day_of_week service.day_of_week not null,
	time_start interval not null,
	duration interval not null,
	time_range tsrange generated always as (
  	tsrange(timestamp '2000-01-01' + time_start, 
  					timestamp '2000-01-01' + time_start + duration)
  ) stored,
	exclude using gist (
  	day_of_week with =,
    time_range with &&
  )
);


create or replace function service.get_free_slots(date_from timestamp with time zone, date_to timestamp with time zone)
returns json
language plpgsql
as $function$
declare
	res json;
begin  
		with days as (
			select * from service.get_days(now(), now() + interval '1 month')
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
	return res;

end;
$function$;

select service.get_free_slots(now(), now() + interval '1 month');


create or replace function service.get_days(from_date timestamp with time zone, to_date timestamp with time zone)
returns table (
	date date,
  day_of_week service.day_of_week
)
language plpgsql
as $function$
declare
	res json;
begin  
	return query
		select d::date,
    			 lower(trim(to_char(d, 'Day')))::service.day_of_week
	  from generate_series(from_date, to_date, interval '1 day') d;
end;
$function$;
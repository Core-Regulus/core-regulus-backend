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

select * from service.meeting_time_slots;




	
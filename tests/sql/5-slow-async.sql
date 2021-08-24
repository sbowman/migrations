# --- !Up /async
select pg_sleep(3);
insert into samples (name) values ('slowup');

# --- !Down /async
select pg_sleep(3);
insert into samples (name) values ('slowdown');

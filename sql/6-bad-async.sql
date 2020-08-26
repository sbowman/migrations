# --- !Up /async
select pg_sleep(3);
insert into samples (blah) values ('noway');

# --- !Down /async
select pg_sleep(3);
insert into samples (blah) values ('noway');

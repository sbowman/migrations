# --- !Up /async
insert into samples (name) values ('aaa');
insert into samples (name) values ('ccc');

# --- !Down /async
delete from samples where name = 'ccc';
delete from samples where name = 'aaa';

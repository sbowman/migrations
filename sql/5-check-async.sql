# --- !Up /notx /async
insert into samples (name) values ('aaa');
insert into samples (blah) values ('bbb');
insert into samples (name) values ('ccc');

# --- !Down /notx /async
delete from samples where name = 'ccc';
delete from samples where name = 'aaa';
delete from samples where blah is not null;

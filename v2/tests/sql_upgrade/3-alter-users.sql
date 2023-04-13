--- !Up
alter table users add column age integer;

--- !Down
alter table users drop column age;

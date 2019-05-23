# --- !Up
alter table sample add column email varchar(1024);
create unique index idx_sample_email on sample (email);

# --- !Down
alter table sample drop column email;

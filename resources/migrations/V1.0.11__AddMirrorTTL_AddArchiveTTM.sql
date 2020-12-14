
alter table backups add mirror_lifetime_in_days smallint CHECK (mirror_lifetime_in_days >= 0);
alter table backups add archive_ttm smallint CHECK (archive_ttm >= 0);

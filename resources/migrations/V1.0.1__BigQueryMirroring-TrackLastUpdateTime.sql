alter table source_metadata
    add last_modified_time timestamp;

update source_metadata
    set last_modified_time = audit_created_timestamp
    where operation != 'Delete';
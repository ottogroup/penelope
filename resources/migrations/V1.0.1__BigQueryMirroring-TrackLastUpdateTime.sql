alter table source_metadata
    add last_modified_time timestamp;

update source_metadata
SET last_modified_time= case
                            when operation != 'Delete' then audit_created_timestamp
    end
;

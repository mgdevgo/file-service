create table if not exists file_meta (
    id uuid primary key,
    filename text not null default '',
    hash text not null default '',
    created_at timestamp with time zone default now(),
    updated_at timestamp with time zone default now()
);

create index idx_file_meta_updated_at_created_at on file_meta (updated_at desc, created_at desc);
create index idx_file_meta_hash on file_meta (hash);
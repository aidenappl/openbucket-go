create table core.authorizations
(
    id           serial
        primary key,
    name         text         not null,
    key_id       varchar(64)  not null
        unique,
    secret_key   varchar(128) not null,
    date_created timestamp default now()
);

alter table core.authorizations
    owner to openbucket;

create table core.buckets
(
    id            serial
        primary key,
    name          text not null
        unique,
    creation_date timestamp     default now(),
    acl           core.acl_type default 'PRIVATE'::core.acl_type,
    owner_id      integer
        references core.authorizations
);

alter table core.buckets
    owner to openbucket;

create table core.bucket_permissions
(
    id         serial
        primary key,
    bucket_id  integer
        references core.buckets,
    grantee_id integer
        references core.authorizations,
    permission core.permission_type not null,
    date_added timestamp default now()
);

alter table core.bucket_permissions
    owner to openbucket;

create table core.bucket_tags
(
    id        serial
        primary key,
    bucket_id integer
        references core.buckets,
    tag_key   text not null,
    tag_value text not null
);

alter table core.bucket_tags
    owner to openbucket;

create table core.objects
(
    id            serial
        primary key,
    bucket_id     integer
        references core.buckets,
    key           text        not null,
    etag          varchar(64) not null,
    version_id    integer default 1,
    owner_id      integer
        references core.authorizations,
    public        boolean default false,
    size          bigint  default 0,
    last_modified timestamp   not null,
    uploaded_at   timestamp   not null
);

alter table core.objects
    owner to openbucket;

create table core.object_tags
(
    id        serial
        primary key,
    bucket_id integer
        references core.buckets,
    object_id integer
        references core.objects,
    tag_key   text not null,
    tag_value text not null
);

alter table core.object_tags
    owner to openbucket;

-- Performance indexes
create index if not exists idx_objects_bucket_key on core.objects(bucket_id, key);
create index if not exists idx_object_tags_bucket_object on core.object_tags(bucket_id, object_id);
create index if not exists idx_bucket_permissions_bucket_grantee on core.bucket_permissions(bucket_id, grantee_id);
create index if not exists idx_bucket_tags_bucket on core.bucket_tags(bucket_id);

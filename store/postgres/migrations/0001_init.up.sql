SET client_encoding = 'UTF8';

--CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;
--COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;
COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';

-- Create the users table
CREATE TABLE IF NOT EXISTS public.user (
  id TEXT PRIMARY KEY DEFAULT public.uuid_generate_v4(),
  login TEXT NOT NULL,
  address TEXT NOT NULL DEFAULT '',
  balance bigint DEFAULT 0
);

CREATE UNIQUE INDEX uix_user_login ON public.user USING btree (login);
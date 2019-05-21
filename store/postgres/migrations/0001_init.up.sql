SET client_encoding = 'UTF8';

--CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;
--COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;
COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';

-- users table
CREATE TABLE public.user (
  id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
  login TEXT NOT NULL,
  address TEXT NOT NULL DEFAULT '',
  balance bigint DEFAULT 0
);

CREATE UNIQUE INDEX uix_user_login ON public.user USING btree (login);

-- invoice table
CREATE TABLE public.invoice (
  id uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
  user_id uuid NOT NULL,
  payment_hash TEXT NOT NULL,
  status TEXT
);

CREATE UNIQUE INDEX uix_invoice_payment_hash ON public.invoice USING btree (payment_hash);
CREATE INDEX ix_invoice_user_id_status ON public.invoice USING btree(user_id, status);

ALTER TABLE ONLY public.invoice
  ADD CONSTRAINT fkey_invoice_user_id FOREIGN KEY (user_id) REFERENCES public.user(id) ON DELETE CASCADE;

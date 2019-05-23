SET client_encoding = 'UTF8';

--CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;
--COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';

-- account table
CREATE TABLE public.account (
  id TEXT PRIMARY KEY, -- type:id
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE,
  address TEXT NOT NULL DEFAULT '',
  balance BIGINT DEFAULT 0,
  balance_in BIGINT DEFAULT 0,
  balance_out BIGINT DEFAULT 0
);

-- Create the unknown account that can be used to handle unaccounted for transactions
INSERT INTO account (id, updated_at, address) VALUES ('internal:unknown', NOW(), 'unknown');

-- ledger types
CREATE TYPE ledger_status AS ENUM ('pending', 'completed', 'expired');
CREATE TYPE ledger_type AS ENUM ('btc', 'lightning');
CREATE TYPE ledger_direction AS ENUM ('in', 'out');

-- ledger table
CREATE TABLE public.ledger (
  id TEXT PRIMARY KEY,  -- This will be the invoice/payreq payment_hash or txid
  account_id TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE,
  expires_at TIMESTAMP WITH TIME ZONE,
  status ledger_status,
  type ledger_type,
  direction ledger_direction,
  value BIGINT DEFAULT 0,
  memo text,
  request text
);

ALTER TABLE ONLY public.ledger
  ADD CONSTRAINT fkey_ledger_account_id FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;

-- Our default listing format
CREATE INDEX ix_ledger_account_id_status_updated_at ON public.ledger USING btree(account_id, status, updated_at);

-- System wide stats lookup
CREATE INDEX ix_ledger_stats ON public.ledger USING btree(type, direction, status, request, account_id);

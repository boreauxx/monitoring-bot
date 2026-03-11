package postgres

const storeAssetQuery = `
INSERT INTO public.assets (name,
                           address,
                           interval_seconds,
                           timeout_seconds)
VALUES ($1, $2, $3, $4)
ON CONFLICT (address) 
    DO UPDATE 
SET name = excluded.name,
    interval_seconds = excluded.interval_seconds,
    timeout_seconds = excluded.timeout_seconds,
    updated_at = NOW()
RETURNING 
    id, 
    name,
    address,
    interval_seconds,
    timeout_seconds,
    created_at,
    updated_at`

const storeProbeQuery = `
INSERT INTO public.probe_events (success,
                                 code,
                                 err_message,
                                 asset_id)
VALUES ($1, $2, $3, $4)
RETURNING 
    id,
    success,
    code,
    err_message,
    created_at,
    asset_id`

const storeIncidentQuery = `
INSERT INTO public.incidents (severity,
                              summary,
                              started_at,
                              ended_at,
                              asset_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING
	id,
	severity,
	summary,
	started_at,
	ended_at,
	asset_id`

const getOpenIncidentByAssetQuery = `
SELECT id,
       severity,
       summary,
       started_at,
       ended_at,
       asset_id
FROM public.incidents
WHERE asset_id = $1
  	AND ended_at IS NULL
ORDER BY started_at DESC
LIMIT 1
FOR UPDATE`

const resolveIncidentQuery = `
UPDATE public.incidents
SET ended_at = $2
WHERE id = $1
RETURNING
    id,
    severity,
    summary,
    started_at,
    ended_at,
    asset_id`

const cleanupProbesQuery = `
DELETE FROM public.probe_events
WHERE created_at <= $1`

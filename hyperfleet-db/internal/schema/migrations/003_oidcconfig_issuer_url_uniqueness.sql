-- Unmanaged OidcConfigs carry a customer-supplied issuerUrl, and platform-api
-- rejects duplicates within an account for a clean user-facing error. That
-- check is a List-then-Create in application code and is not atomic, so two
-- concurrent creates for the same issuerUrl could both pass the check and
-- both succeed. Enforce uniqueness here as the atomic safety net behind the
-- platform-api's application-level collision check (same pattern as
-- idx_cluster_name_hash4 in 002_cluster_dns_uniqueness.sql).
CREATE UNIQUE INDEX IF NOT EXISTS idx_oidcconfig_unmanaged_issuer_url
    ON kubernetes_resources (namespace, (spec->>'issuerUrl'))
    WHERE gvk = 'hyperfleet.io/v1alpha1/OidcConfig'
      AND deletion_timestamp IS NULL
      AND spec->>'type' = 'unmanaged'
      AND spec->>'issuerUrl' IS NOT NULL;

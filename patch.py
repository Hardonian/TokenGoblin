import sys

def patch_file(path, old_str, new_str):
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    content = content.replace(old_str, new_str)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)

old_interface = """	SaveAPIKey(ctx context.Context, key domain.APIKey) error
	GetAPIKey(ctx context.Context, keyID string) (*domain.APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error
	UpsertTenantMember(ctx context.Context, member domain.TenantMember) error"""

new_interface = """	SaveAPIKey(ctx context.Context, key domain.APIKey) error
	GetAPIKey(ctx context.Context, keyID string) (*domain.APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error
	ListAPIKeys(ctx context.Context, tenantID string) ([]domain.APIKey, error)
	RevokeAPIKey(ctx context.Context, keyID string, tenantID string) error
	UpsertTenantMember(ctx context.Context, member domain.TenantMember) error"""

old_unavailable = """func (r *UnavailableRepository) UpdateAPIKeyLastUsed(context.Context, string) error {
	return r.err()
}

func (r *UnavailableRepository) UpsertTenantMember(context.Context, domain.TenantMember) error {"""

new_unavailable = """func (r *UnavailableRepository) UpdateAPIKeyLastUsed(context.Context, string) error {
	return r.err()
}

func (r *UnavailableRepository) ListAPIKeys(context.Context, string) ([]domain.APIKey, error) {
	return nil, r.err()
}

func (r *UnavailableRepository) RevokeAPIKey(context.Context, string, string) error {
	return r.err()
}

func (r *UnavailableRepository) UpsertTenantMember(context.Context, domain.TenantMember) error {"""

patch_file('internal/storage/repository.go', old_interface, new_interface)
patch_file('internal/storage/repository.go', old_unavailable, new_unavailable)
print("Patched repository.go")

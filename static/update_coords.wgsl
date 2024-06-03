@group(0) @binding(0) var<storage, read_write> coords: array<vec2<f32>>;
@group(0) @binding(1) var<storage> dist_matrix: array<f32>;
@group(0) @binding(2) var<uniform> params: Params;

struct Params {
    pivot_idx: u32,
    learning_rate: f32,
    N: u32,
}

@compute @workgroup_size(64)
fn update_coords(@builtin(global_invocation_id) id: vec3<u32>) {
    let rij = dist_matrix[params.pivot_idx * params.N + id.x];
    let diff = coords[params.pivot_idx] - coords[id.x];
    let dij = length(diff);

    coords[id.x] -= params.learning_rate * (rij - dij) * (diff / (dij + 1e-6));
}

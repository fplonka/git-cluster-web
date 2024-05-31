@group(0) @binding(0) var<storage, read_write> coords: array<vec2<f32>>;
@group(0) @binding(1) var<storage> dist_matrix: array<f32>;
@group(0) @binding(2) var<uniform> params: Params;

struct Params {
    pivot_idx: u32,
    learning_rate: f32,
    N: u32,
}

@compute @workgroup_size(1)
fn update_coords(@builtin(global_invocation_id) id: vec3<u32>) {
    let j = id.x;
    if j >= params.N {
        return;
    }

    let xi = coords[params.pivot_idx];
    let xj = coords[j];
    let rij = dist_matrix[params.pivot_idx * params.N + j];

    let dij = sqrt((xi.x - xj.x) * (xi.x - xj.x) + (xi.y - xj.y) * (xi.y - xj.y));
    coords[j] -= (xi - xj) * params.learning_rate * (rij - dij) / (dij + 1e-6);
}
@group(0) @binding(0) var<storage, read> entries: array<i32>;
@group(0) @binding(1) var<storage, read> startIndices: array<i32>;
@group(0) @binding(2) var<storage, read_write> distMatrix: array<f32>;

fn jaccard(row1Start: i32, row2Start: i32) -> f32 {
    var intersection: i32 = 0;
    var unionSize: i32 = 0;
    var i: i32 = row1Start;
    var j: i32 = row2Start;

    let paddingValue: i32 = -1;

    loop {
        if entries[i] == paddingValue && entries[j] == paddingValue {
            break;
        }

        if entries[i] == entries[j] {
            if entries[i] == paddingValue {
                break;
            }
            intersection += 1;
            i += 1;
            j += 1;
        } else if entries[i] < entries[j] || entries[j] == paddingValue {
            if entries[i] == paddingValue {
                break;
            }
            i += 1;
        } else {
            if entries[j] == paddingValue {
                break;
            }
            j += 1;
        }
        unionSize += 1;
    }

    // Add remaining elements in row1 and row2 for union count
    while entries[i] != paddingValue {
        unionSize += 1;
        i += 1;
    }
    while entries[j] != paddingValue {
        unionSize += 1;
        j += 1;
    }

    return f32(intersection) / f32(unionSize);
}

@compute @workgroup_size(8, 8, 1)
fn main(@builtin(global_invocation_id) global_id: vec3<u32>) {
    let N: u32 = arrayLength(&startIndices);
    let index: u32 = N * global_id.x + global_id.y;

    if index >= N * N {
        return;
    }

    let i: u32 = index / N;
    let j: u32 = index % N;

    if i == j {
        distMatrix[i * N + j] = 0.0;
    } else {
        let row1Start: i32 = startIndices[i];
        let row2Start: i32 = startIndices[j];
        let score: f32 = jaccard(row1Start, row2Start);
        let distance: f32 = 1.0 - score + 1e-8;
        distMatrix[i * N + j] = distance;
        distMatrix[j * N + i] = distance;
    }
}

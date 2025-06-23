document.getElementById('sourceIdForm').addEventListener('submit', function(e) {
    e.preventDefault();
    
    const info = {
        package: document.getElementById('package').value,
        domain: document.getElementById('domain').value,
        language: document.getElementById('language').value,
        nsfw: document.getElementById('nsfw').checked,
        version: document.getElementById('version').value
    };

    // Clean domain
    let domain = info.domain
        .replace('https://', '')
        .replace('http://', '')
        .replace('www.', '');

    // Create base info string
    const baseInfo = [
        info.package.toLowerCase(),
        domain.toLowerCase(),
        info.language.toLowerCase(),
        info.nsfw ? '1' : '0',
        info.version
    ].join(':');

    // Generate deterministic hash using FNV-1a
    function fnv1a(str) {
        const FNV_PRIME = 0x01000193;
        const FNV_OFFSET_BASIS = 0x811c9dc5;
        
        let hash = FNV_OFFSET_BASIS;
        for (let i = 0; i < str.length; i++) {
            hash ^= str.charCodeAt(i);
            hash = Math.imul(hash, FNV_PRIME);
        }
        return hash >>> 0; // Convert to unsigned 32-bit
    }

    // Generate random component
    const randomBytes = new Uint32Array(1);
    crypto.getRandomValues(randomBytes);
    const randomComponent = randomBytes[0];

    // Combine them using BigInt for 64-bit operations
    const baseHash = BigInt(fnv1a(baseInfo));
    const sourceId = (baseHash << 32n) | BigInt(randomComponent);

    // Display the result
    document.getElementById('result').classList.remove('hidden');
    document.getElementById('sourceId').textContent = sourceId.toString();
}); 
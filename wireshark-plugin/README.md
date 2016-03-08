# Usage

Run `wireshark -X lua_script:suft.lua`.

# Install

Copy `suft.lua` to wireshark/plugins/.

Append `dofile(DATA_DIR.."/plugins/suft.lua")` at the end of wireshark/init.lua.

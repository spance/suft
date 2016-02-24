-- create suft_p protocol and its fields
suft_p = Proto.new("suft","SUFT Protocol")

-- ProtoField.new(name, abbr, type, [voidstring], [base], [mask], [descr])
local f_margic = ProtoField.new("Magic Header", "suft.magic", ftypes.BYTES)
local f_len = ProtoField.new("Length", "suft.len", ftypes.UINT16)
local f_rcid = ProtoField.new("Remote ConnId", "suft.rcid", ftypes.UINT32)
local f_lcid = ProtoField.new("Local ConnId", "suft.lcid", ftypes.UINT32)
local f_seq = ProtoField.new("Seq", "suft.seq", ftypes.UINT32)
local f_ack = ProtoField.new("Ack", "suft.ack", ftypes.UINT32)
local f_flags = ProtoField.new("Flags", "suft.flags", ftypes.UINT8, nil, base.HEX)
local f_flag_syn = ProtoField.new("SYN", "suft.flags.syn", ftypes.UINT8, nil, base.HEX, 1)
local f_flag_ack = ProtoField.new("ACK", "suft.flags.ack", ftypes.UINT8, nil, base.HEX, 2)
local f_flag_sack = ProtoField.new("SACK", "suft.flags.sack", ftypes.UINT8, nil, base.HEX, 4)
local f_flag_time = ProtoField.new("TIME", "suft.flags.time", ftypes.UINT8, nil, base.HEX, 8)
local f_flag_data = ProtoField.new("DATA", "suft.flags.data", ftypes.UINT8, nil, base.HEX, 16)
local f_flag_reset = ProtoField.new("RESET", "suft.flags.reset", ftypes.UINT8, nil, base.HEX, 64)
local f_flag_fin = ProtoField.new("FIN", "suft.flags.fin", ftypes.UINT8, nil, base.HEX, 128)
local f_scnt = ProtoField.new("Sent Count", "suft.scnt", ftypes.UINT8)
local f_payload = ProtoField.new("Payload", "suft.payload", ftypes.BYTES)

local f_sack_tbl = ProtoField.new("SACK TBL", "suft.sack.tbl", ftypes.UINT8)
local f_sack_scnt = ProtoField.new("SACK Sent Count", "suft.sack.scnt", ftypes.UINT8)
local f_sack_delayed = ProtoField.new("SACK Delayed", "suft.sack.delayed", ftypes.UINT16)
local f_sack_bitmap = ProtoField.new("SACK Bitmap", "suft.sack.bitmap", ftypes.UINT64, nil, base.HEX)

suft_p.fields = {
	f_margic,f_len,f_rcid,f_lcid,f_seq,f_ack,f_flags,
	f_flag_syn,f_flag_ack,f_flag_sack,f_flag_time,f_flag_data,f_flag_reset,f_flag_fin,
	f_scnt,f_payload,f_sack_tbl,f_sack_scnt,f_sack_delayed,f_sack_bitmap,
}

local packet_type = {
	[1] = "SYN",
	[2] = "ACK",
	[3] = "SYN+ACK",
	[4] = "SACK",
	[12] = "SACK+TIME",
	[16] = "DATA",
	[128] = "FIN",
}

-- suft_p dissector function
function suft_p.dissector (buf, pinfo, root)
	-- validate packet length is adequate, otherwise quit
	if buf:len() < 26 then return end
	pinfo.cols.protocol = suft_p.name

	-- create subtree for suft_p
	local subtree = root:add(suft_p, buf())
	-- add protocol fields to subtree
	subtree:add(f_margic, buf(0,6))
	subtree:add(f_len, buf(6,2))
	subtree:add(f_rcid, buf(8,4))
	subtree:add(f_lcid, buf(12,4))
	local v_seq, v_ack = buf(16,4), buf(20,4)
	local t_seq = subtree:add(f_seq, v_seq)
	subtree:add(f_ack, v_ack)

	local flags = buf(24,1)
	local ft = subtree:add(f_flags, flags)
	ft:add(f_flag_syn, flags)
	ft:add(f_flag_ack, flags)
	ft:add(f_flag_sack, flags)
	ft:add(f_flag_time, flags)
	ft:add(f_flag_data, flags)
	ft:add(f_flag_reset, flags)
	ft:add(f_flag_fin, flags)
	local v_flag = flags:uint()
	local ptype = packet_type[v_flag]
	local v_scnt = buf(25,1)
	if ptype then
		local fmt = "[%s] seq=%u ack=%u scnt=%u"
		if v_flag == 1 then
			fmt = fmt .. " ###### New Connection ######"
		end
		pinfo.cols.info = string.format(fmt, ptype, v_seq:uint(), v_ack:uint(), v_scnt:uint())
		ft:append_text(string.format(" [%s]", ptype))
	end

	subtree:add(f_scnt, v_scnt)
	local payload = subtree:add(f_payload, buf(26))
	if bit32.btest(v_flag, 4) then -- sack
		t_seq:append_text(" [TRP]")
		local sack_tbl,sack_scnt,sack_delayed = buf(26,1),buf(27,1),buf(28,2)
		payload:add(f_sack_tbl, sack_tbl)
		payload:add(f_sack_scnt, sack_scnt)
		payload:add(f_sack_delayed, sack_delayed)
		local i, max_i = 0, buf:len() - 8
		for i=30,max_i,8 do
			payload:add(f_sack_bitmap, buf(i,8)):append_text(string.format(" [%u]", (i-30)/8))
		end
	elseif bit32.btest(v_flag, 16) then -- data
		payload:append_text(string.format(" [Len %u]", buf:len()-26))
	end
end


local dissector_table = DissectorTable.get("udp.port")
dissector_table:add(9090, suft_p)
dissector_table:add(9008, suft_p)
dissector_table:add(5001, suft_p)

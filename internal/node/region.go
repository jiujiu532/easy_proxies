package node

import (
	"strings"
)

// RegionInfo contains region details
type RegionInfo struct {
	Code string // ISO 3166-1 alpha-2 code (e.g., "US", "JP")
	Name string // Full name (e.g., "United States", "Japan")
	Flag string // Emoji flag
}


// regionPatterns maps keywords to region info
var regionPatterns = map[string]RegionInfo{
	// East Asia
	"hk":        {Code: "HK", Name: "Hong Kong", Flag: "🇭🇰"},
	"hongkong":  {Code: "HK", Name: "Hong Kong", Flag: "🇭🇰"},
	"hong kong": {Code: "HK", Name: "Hong Kong", Flag: "🇭🇰"},
	"香港":        {Code: "HK", Name: "Hong Kong", Flag: "🇭🇰"},

	"tw":      {Code: "TW", Name: "Taiwan", Flag: "🇹🇼"},
	"taiwan":  {Code: "TW", Name: "Taiwan", Flag: "🇹🇼"},
	"台湾":      {Code: "TW", Name: "Taiwan", Flag: "🇹🇼"},
	"台灣":      {Code: "TW", Name: "Taiwan", Flag: "🇹🇼"},

	"jp":     {Code: "JP", Name: "Japan", Flag: "🇯🇵"},
	"japan":  {Code: "JP", Name: "Japan", Flag: "🇯🇵"},
	"日本":     {Code: "JP", Name: "Japan", Flag: "🇯🇵"},
	"东京":     {Code: "JP", Name: "Japan", Flag: "🇯🇵"},
	"大阪":     {Code: "JP", Name: "Japan", Flag: "🇯🇵"},

	"kr":     {Code: "KR", Name: "South Korea", Flag: "🇰🇷"},
	"korea":  {Code: "KR", Name: "South Korea", Flag: "🇰🇷"},
	"韩国":     {Code: "KR", Name: "South Korea", Flag: "🇰🇷"},
	"韓國":     {Code: "KR", Name: "South Korea", Flag: "🇰🇷"},
	"首尔":     {Code: "KR", Name: "South Korea", Flag: "🇰🇷"},

	"cn":    {Code: "CN", Name: "China", Flag: "🇨🇳"},
	"china": {Code: "CN", Name: "China", Flag: "🇨🇳"},
	"中国":    {Code: "CN", Name: "China", Flag: "🇨🇳"},
	"中國":    {Code: "CN", Name: "China", Flag: "🇨🇳"},
	"上海":    {Code: "CN", Name: "China", Flag: "🇨🇳"},
	"北京":    {Code: "CN", Name: "China", Flag: "🇨🇳"},
	"广州":    {Code: "CN", Name: "China", Flag: "🇨🇳"},
	"深圳":    {Code: "CN", Name: "China", Flag: "🇨🇳"},

	// Southeast Asia
	"sg":        {Code: "SG", Name: "Singapore", Flag: "🇸🇬"},
	"singapore": {Code: "SG", Name: "Singapore", Flag: "🇸🇬"},
	"新加坡":       {Code: "SG", Name: "Singapore", Flag: "🇸🇬"},
	"狮城":        {Code: "SG", Name: "Singapore", Flag: "🇸🇬"},

	"my":       {Code: "MY", Name: "Malaysia", Flag: "🇲🇾"},
	"malaysia": {Code: "MY", Name: "Malaysia", Flag: "🇲🇾"},
	"马来西亚":     {Code: "MY", Name: "Malaysia", Flag: "🇲🇾"},

	"th":       {Code: "TH", Name: "Thailand", Flag: "🇹🇭"},
	"thailand": {Code: "TH", Name: "Thailand", Flag: "🇹🇭"},
	"泰国":       {Code: "TH", Name: "Thailand", Flag: "🇹🇭"},
	"曼谷":       {Code: "TH", Name: "Thailand", Flag: "🇹🇭"},

	"vn":       {Code: "VN", Name: "Vietnam", Flag: "🇻🇳"},
	"vietnam":  {Code: "VN", Name: "Vietnam", Flag: "🇻🇳"},
	"越南":       {Code: "VN", Name: "Vietnam", Flag: "🇻🇳"},

	"ph":          {Code: "PH", Name: "Philippines", Flag: "🇵🇭"},
	"philippines": {Code: "PH", Name: "Philippines", Flag: "🇵🇭"},
	"菲律宾":         {Code: "PH", Name: "Philippines", Flag: "🇵🇭"},

	"id":        {Code: "ID", Name: "Indonesia", Flag: "🇮🇩"},
	"indonesia": {Code: "ID", Name: "Indonesia", Flag: "🇮🇩"},
	"印尼":        {Code: "ID", Name: "Indonesia", Flag: "🇮🇩"},
	"印度尼西亚":     {Code: "ID", Name: "Indonesia", Flag: "🇮🇩"},

	// North America
	"us":      {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"usa":     {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"america": {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"美国":      {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"洛杉矶":     {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"纽约":      {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"西雅图":     {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"圣何塞":     {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"硅谷":      {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"达拉斯":     {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"芝加哥":     {Code: "US", Name: "United States", Flag: "🇺🇸"},

	"ca":     {Code: "CA", Name: "Canada", Flag: "🇨🇦"},
	"canada": {Code: "CA", Name: "Canada", Flag: "🇨🇦"},
	"加拿大":    {Code: "CA", Name: "Canada", Flag: "🇨🇦"},
	"多伦多":    {Code: "CA", Name: "Canada", Flag: "🇨🇦"},
	"温哥华":    {Code: "CA", Name: "Canada", Flag: "🇨🇦"},

	// Europe
	"uk":      {Code: "GB", Name: "United Kingdom", Flag: "🇬🇧"},
	"gb":      {Code: "GB", Name: "United Kingdom", Flag: "🇬🇧"},
	"england": {Code: "GB", Name: "United Kingdom", Flag: "🇬🇧"},
	"britain": {Code: "GB", Name: "United Kingdom", Flag: "🇬🇧"},
	"英国":      {Code: "GB", Name: "United Kingdom", Flag: "🇬🇧"},
	"伦敦":      {Code: "GB", Name: "United Kingdom", Flag: "🇬🇧"},

	"de":      {Code: "DE", Name: "Germany", Flag: "🇩🇪"},
	"germany": {Code: "DE", Name: "Germany", Flag: "🇩🇪"},
	"德国":      {Code: "DE", Name: "Germany", Flag: "🇩🇪"},
	"法兰克福":    {Code: "DE", Name: "Germany", Flag: "🇩🇪"},

	"fr":     {Code: "FR", Name: "France", Flag: "🇫🇷"},
	"france": {Code: "FR", Name: "France", Flag: "🇫🇷"},
	"法国":     {Code: "FR", Name: "France", Flag: "🇫🇷"},
	"巴黎":     {Code: "FR", Name: "France", Flag: "🇫🇷"},

	"nl":          {Code: "NL", Name: "Netherlands", Flag: "🇳🇱"},
	"netherlands": {Code: "NL", Name: "Netherlands", Flag: "🇳🇱"},
	"荷兰":          {Code: "NL", Name: "Netherlands", Flag: "🇳🇱"},
	"阿姆斯特丹":       {Code: "NL", Name: "Netherlands", Flag: "🇳🇱"},

	"ru":     {Code: "RU", Name: "Russia", Flag: "🇷🇺"},
	"russia": {Code: "RU", Name: "Russia", Flag: "🇷🇺"},
	"俄罗斯":    {Code: "RU", Name: "Russia", Flag: "🇷🇺"},
	"莫斯科":    {Code: "RU", Name: "Russia", Flag: "🇷🇺"},

	"it":    {Code: "IT", Name: "Italy", Flag: "🇮🇹"},
	"italy": {Code: "IT", Name: "Italy", Flag: "🇮🇹"},
	"意大利":   {Code: "IT", Name: "Italy", Flag: "🇮🇹"},

	"es":    {Code: "ES", Name: "Spain", Flag: "🇪🇸"},
	"spain": {Code: "ES", Name: "Spain", Flag: "🇪🇸"},
	"西班牙":   {Code: "ES", Name: "Spain", Flag: "🇪🇸"},

	"ch":          {Code: "CH", Name: "Switzerland", Flag: "🇨🇭"},
	"switzerland": {Code: "CH", Name: "Switzerland", Flag: "🇨🇭"},
	"瑞士":          {Code: "CH", Name: "Switzerland", Flag: "🇨🇭"},

	"se":     {Code: "SE", Name: "Sweden", Flag: "🇸🇪"},
	"sweden": {Code: "SE", Name: "Sweden", Flag: "🇸🇪"},
	"瑞典":     {Code: "SE", Name: "Sweden", Flag: "🇸🇪"},

	"fi":      {Code: "FI", Name: "Finland", Flag: "🇫🇮"},
	"finland": {Code: "FI", Name: "Finland", Flag: "🇫🇮"},
	"芬兰":      {Code: "FI", Name: "Finland", Flag: "🇫🇮"},

	"no":     {Code: "NO", Name: "Norway", Flag: "🇳🇴"},
	"norway": {Code: "NO", Name: "Norway", Flag: "🇳🇴"},
	"挪威":     {Code: "NO", Name: "Norway", Flag: "🇳🇴"},

	"pl":     {Code: "PL", Name: "Poland", Flag: "🇵🇱"},
	"poland": {Code: "PL", Name: "Poland", Flag: "🇵🇱"},
	"波兰":     {Code: "PL", Name: "Poland", Flag: "🇵🇱"},

	"tr":     {Code: "TR", Name: "Turkey", Flag: "🇹🇷"},
	"turkey": {Code: "TR", Name: "Turkey", Flag: "🇹🇷"},
	"土耳其":    {Code: "TR", Name: "Turkey", Flag: "🇹🇷"},

	// Oceania
	"au":        {Code: "AU", Name: "Australia", Flag: "🇦🇺"},
	"australia": {Code: "AU", Name: "Australia", Flag: "🇦🇺"},
	"澳大利亚":      {Code: "AU", Name: "Australia", Flag: "🇦🇺"},
	"悉尼":        {Code: "AU", Name: "Australia", Flag: "🇦🇺"},
	"墨尔本":       {Code: "AU", Name: "Australia", Flag: "🇦🇺"},

	"nz":           {Code: "NZ", Name: "New Zealand", Flag: "🇳🇿"},
	"new zealand":  {Code: "NZ", Name: "New Zealand", Flag: "🇳🇿"},
	"newzealand":   {Code: "NZ", Name: "New Zealand", Flag: "🇳🇿"},
	"新西兰":         {Code: "NZ", Name: "New Zealand", Flag: "🇳🇿"},

	// South America
	"br":     {Code: "BR", Name: "Brazil", Flag: "🇧🇷"},
	"brazil": {Code: "BR", Name: "Brazil", Flag: "🇧🇷"},
	"巴西":     {Code: "BR", Name: "Brazil", Flag: "🇧🇷"},

	"ar":        {Code: "AR", Name: "Argentina", Flag: "🇦🇷"},
	"argentina": {Code: "AR", Name: "Argentina", Flag: "🇦🇷"},
	"阿根廷":       {Code: "AR", Name: "Argentina", Flag: "🇦🇷"},

	// Middle East
	"ae":  {Code: "AE", Name: "UAE", Flag: "🇦🇪"},
	"uae": {Code: "AE", Name: "UAE", Flag: "🇦🇪"},
	"阿联酋": {Code: "AE", Name: "UAE", Flag: "🇦🇪"},
	"迪拜":  {Code: "AE", Name: "UAE", Flag: "🇦🇪"},

	"il":     {Code: "IL", Name: "Israel", Flag: "🇮🇱"},
	"israel": {Code: "IL", Name: "Israel", Flag: "🇮🇱"},
	"以色列":    {Code: "IL", Name: "Israel", Flag: "🇮🇱"},

	// South Asia
	"in":    {Code: "IN", Name: "India", Flag: "🇮🇳"},
	"india": {Code: "IN", Name: "India", Flag: "🇮🇳"},
	"印度":    {Code: "IN", Name: "India", Flag: "🇮🇳"},
}

// flagToRegion maps emoji flags to region codes
var flagToRegion = map[string]RegionInfo{
	"🇭🇰": {Code: "HK", Name: "Hong Kong", Flag: "🇭🇰"},
	"🇹🇼": {Code: "TW", Name: "Taiwan", Flag: "🇹🇼"},
	"🇯🇵": {Code: "JP", Name: "Japan", Flag: "🇯🇵"},
	"🇰🇷": {Code: "KR", Name: "South Korea", Flag: "🇰🇷"},
	"🇨🇳": {Code: "CN", Name: "China", Flag: "🇨🇳"},
	"🇸🇬": {Code: "SG", Name: "Singapore", Flag: "🇸🇬"},
	"🇲🇾": {Code: "MY", Name: "Malaysia", Flag: "🇲🇾"},
	"🇹🇭": {Code: "TH", Name: "Thailand", Flag: "🇹🇭"},
	"🇻🇳": {Code: "VN", Name: "Vietnam", Flag: "🇻🇳"},
	"🇵🇭": {Code: "PH", Name: "Philippines", Flag: "🇵🇭"},
	"🇮🇩": {Code: "ID", Name: "Indonesia", Flag: "🇮🇩"},
	"🇺🇸": {Code: "US", Name: "United States", Flag: "🇺🇸"},
	"🇨🇦": {Code: "CA", Name: "Canada", Flag: "🇨🇦"},
	"🇬🇧": {Code: "GB", Name: "United Kingdom", Flag: "🇬🇧"},
	"🇩🇪": {Code: "DE", Name: "Germany", Flag: "🇩🇪"},
	"🇫🇷": {Code: "FR", Name: "France", Flag: "🇫🇷"},
	"🇳🇱": {Code: "NL", Name: "Netherlands", Flag: "🇳🇱"},
	"🇷🇺": {Code: "RU", Name: "Russia", Flag: "🇷🇺"},
	"🇮🇹": {Code: "IT", Name: "Italy", Flag: "🇮🇹"},
	"🇪🇸": {Code: "ES", Name: "Spain", Flag: "🇪🇸"},
	"🇨🇭": {Code: "CH", Name: "Switzerland", Flag: "🇨🇭"},
	"🇸🇪": {Code: "SE", Name: "Sweden", Flag: "🇸🇪"},
	"🇫🇮": {Code: "FI", Name: "Finland", Flag: "🇫🇮"},
	"🇳🇴": {Code: "NO", Name: "Norway", Flag: "🇳🇴"},
	"🇵🇱": {Code: "PL", Name: "Poland", Flag: "🇵🇱"},
	"🇹🇷": {Code: "TR", Name: "Turkey", Flag: "🇹🇷"},
	"🇦🇺": {Code: "AU", Name: "Australia", Flag: "🇦🇺"},
	"🇳🇿": {Code: "NZ", Name: "New Zealand", Flag: "🇳🇿"},
	"🇧🇷": {Code: "BR", Name: "Brazil", Flag: "🇧🇷"},
	"🇦🇷": {Code: "AR", Name: "Argentina", Flag: "🇦🇷"},
	"🇦🇪": {Code: "AE", Name: "UAE", Flag: "🇦🇪"},
	"🇮🇱": {Code: "IL", Name: "Israel", Flag: "🇮🇱"},
	"🇮🇳": {Code: "IN", Name: "India", Flag: "🇮🇳"},
}

// DetectRegion attempts to identify the region from node name or URI
func DetectRegion(nodeName string) RegionInfo {
	if nodeName == "" {
		return RegionInfo{}
	}

	nameLower := strings.ToLower(nodeName)

	// First, check for emoji flags by direct string matching
	for flag, info := range flagToRegion {
		if strings.Contains(nodeName, flag) {
			return info
		}
	}

	// Then check for keywords
	for keyword, info := range regionPatterns {
		if strings.Contains(nameLower, keyword) {
			return info
		}
	}

	return RegionInfo{}
}


// GetAllRegions returns a list of all known regions
func GetAllRegions() []RegionInfo {
	seen := make(map[string]bool)
	var result []RegionInfo

	for _, info := range regionPatterns {
		if !seen[info.Code] {
			seen[info.Code] = true
			result = append(result, info)
		}
	}

	return result
}

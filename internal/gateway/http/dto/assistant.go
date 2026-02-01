package dto

type AssistantChatHTTPReq struct {
	Message string `json:"message,required"`
	Mode    string `json:"mode"`
	UseRAG  bool   `json:"use_rag"`
	Debug   bool   `json:"debug"`
}

type AssistantIntent struct {
	Name       string            `json:"name"`
	Confidence float64           `json:"confidence"`
	Slots      map[string]string `json:"slots"`
}

type AssistantRAGSnippet struct {
	DocKey  string `json:"doc_key"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type AssistantChatHTTPResp struct {
	Code      int32                `json:"code"`
	Msg       string               `json:"msg"`
	Reply     string               `json:"reply"`
	Intent    *AssistantIntent     `json:"intent,omitempty"`
	ToolName  string               `json:"tool_name,omitempty"`
	ToolData  any                  `json:"tool_data,omitempty"`
	RAGSources []AssistantRAGSnippet `json:"rag_sources,omitempty"`
}

type AssistantKBUpsertHTTPReq struct {
	DocKey  string `json:"doc_key,required"`
	Title   string `json:"title,required"`
	Content string `json:"content,required"`
	Tags    string `json:"tags"`
}

type AssistantKBSearchHTTPReq struct {
	Query string `json:"query,required"`
	Limit int    `json:"limit"`
}

type AssistantKBSearchItem struct {
	DocKey  string `json:"doc_key"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Tags    string `json:"tags"`
}

type AssistantKBSearchHTTPResp struct {
	Code  int32                  `json:"code"`
	Msg   string                 `json:"msg"`
	Items []AssistantKBSearchItem `json:"items"`
}

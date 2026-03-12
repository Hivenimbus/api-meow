<template>
  <div class="p-8 max-w-7xl mx-auto space-y-8 animate-fade-in">
    <!-- Header -->
    <div class="space-y-2">
      <h1 class="text-3xl font-bold text-white tracking-tight">Documentação da API</h1>
      <p class="text-slate-400 text-lg">Guia completo para integração e uso da API Meow</p>
    </div>

    <!-- Navigation Tabs -->
    <div class="flex flex-wrap gap-2 overflow-x-auto pb-2">
      <button 
        v-for="tab in tabs" 
        :key="tab.id"
        @click="activeTab = tab.id"
        :class="[
          'px-4 py-2 rounded-xl text-sm font-medium transition-all whitespace-nowrap',
          activeTab === tab.id 
            ? 'bg-gradient-to-r from-emerald-500 to-teal-500 text-white shadow-lg shadow-emerald-500/20' 
            : 'bg-slate-800/50 text-slate-400 hover:text-white hover:bg-slate-800'
        ]"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Content Area -->
    <div class="grid grid-cols-1 lg:grid-cols-4 gap-8">
      <!-- Main Content -->
      <div class="lg:col-span-3 space-y-8">
        
        <!-- Authentication Section -->
        <section v-if="activeTab === 'auth'" class="space-y-6">
          <div class="bg-slate-800/50 backdrop-blur-sm border border-slate-700/50 rounded-2xl p-6">
            <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
              <span class="w-8 h-8 rounded-lg bg-emerald-500/10 flex items-center justify-center text-emerald-500">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" /></svg>
              </span>
              Autenticação
            </h2>
            <p class="text-slate-300 mb-6 leading-relaxed">
              Todas as requisições para a API devem incluir o cabeçalho <code class="text-emerald-400 bg-emerald-500/10 px-1.5 py-0.5 rounded font-mono text-sm">Authorization</code> com sua chave de API.
            </p>
            
            <div class="bg-slate-950 rounded-xl border border-slate-800 p-4 font-mono text-sm overflow-x-auto">
              <div class="flex items-center justify-between text-slate-500 mb-2 text-xs uppercase tracking-wider">
                <span>Header Example</span>
              </div>
              <div class="text-emerald-400">Authorization: API_KEY_AQUI</div>
            </div>
          </div>
        </section>

        <!-- Instances Section -->
        <section v-if="activeTab === 'instances'" class="space-y-6">
          <EndpointCard 
            method="GET" 
            path="/api/instances" 
            title="Listar Instâncias" 
            description="Retorna todas as instâncias cadastradas." 
          >
            <template #response>
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Minha Instância",
    "status": "connected",
    "phoneNumber": "5511999999999",
    "webhookUrl": "https://api.site.com/webhook",
    "ignoreGroups": true,
    "receiveMessages": true,
    "proxyEnabled": false,
    "createdAt": "2024-01-15T10:30:00Z",
    "updatedAt": "2024-01-15T10:30:00Z"
  }
]
            </template>
          </EndpointCard>
          
          <EndpointCard 
            method="POST" 
            path="/api/instances" 
            title="Criar Instância" 
            description="Cria uma nova instância do WhatsApp."
          >
            <template #body>
{
  "name": "Minha Instância",
  "webhookUrl": "https://api.site.com/webhook",
  "proxyEnabled": true,
  "proxyUrl": "http://user:pass@proxy.com:8080"
}
            </template>
            <template #response>
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Minha Instância",
  "status": "disconnected",
  "webhookUrl": "https://api.site.com/webhook",
  "ignoreGroups": true,
  "receiveMessages": true,
  "proxyEnabled": true,
  "proxyUrl": "http://user:pass@proxy.com:8080",
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-15T10:30:00Z"
}
            </template>
          </EndpointCard>

          <EndpointCard
            method="PUT"
            path="/api/instances/:name/settings"
            title="Atualizar Configurações"
            description="Atualiza configurações de webhook e proxy."
          >
             <template #body>
{
  "webhookUrl": "https://new-url.com",
  "ignoreGroups": true,
  "receiveMessages": true,
  "proxyEnabled": true,
  "proxyUrl": "http://proxy.com:8080"
}
            </template>
            <template #response>
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Minha Instância",
  "status": "connected",
  "phoneNumber": "5511999999999",
  "webhookUrl": "https://new-url.com",
  "ignoreGroups": true,
  "receiveMessages": true,
  "proxyEnabled": true,
  "proxyUrl": "http://proxy.com:8080",
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-15T12:45:00Z"
}
            </template>
          </EndpointCard>

          <EndpointCard
            method="DELETE"
            path="/api/instances/:name"
            title="Deletar Instância"
            description="Remove permanentemente uma instância e desconecta a sessão WhatsApp ativa."
          >
            <template #response>
// 204 No Content — sem corpo de resposta em caso de sucesso

// 404 — Instância não encontrada
{ "error": "Instance not found" }
            </template>
          </EndpointCard>
        </section>

        <!-- Connection Section -->
        <section v-if="activeTab === 'connection'" class="space-y-6">
          <EndpointCard 
            method="POST" 
            path="/api/instances/:name/connect" 
            title="Conectar Instância" 
            description="Inicia a conexão e gera o QR Code se necessário." 
          >
            <template #response>
{
  "status": "connecting",
  "qrCode": "data:image/png;base64,iVBORw0KGgo...",
  "phone": "",
  "message": "Connection initiated"
}
            </template>
          </EndpointCard>
          
          <EndpointCard 
            method="GET" 
            path="/api/instances/:name/wa-status" 
            title="Status da Conexão" 
            description="Verifica se o WhatsApp está conectado." 
          >
            <template #response>
{
  "status": "connected",
  "phone": "5511999999999"
}
            </template>
          </EndpointCard>

          <EndpointCard 
            method="POST" 
            path="/api/instances/:name/disconnect" 
            title="Desconectar Instância" 
            description="Desconecta a sessão ativa do WhatsApp." 
          >
            <template #response>
{
  "message": "Disconnected successfully"
}
            </template>
          </EndpointCard>
        </section>

        <!-- Messages Section -->
        <section v-if="activeTab === 'messages'" class="space-y-6">
          <EndpointCard 
            method="POST" 
            path="/api/instances/:name/send-message" 
            title="Enviar Texto" 
            description="Envia uma mensagem de texto simples." 
          >
            <template #body>
{
  "to": "5511999999999",
  "text": "Olá mundo!",
  "simulateTyping": true,
  "typingDuration": 2000
}
            </template>
            <template #response>
{
  "messageId": "3EB0C767D097B7C09D70",
  "timestamp": "2024-01-15T14:30:00Z"
}
            </template>
          </EndpointCard>

          <EndpointCard 
            method="POST" 
            path="/api/instances/:name/send-media" 
            title="Enviar Mídia" 
            description="Envia imagem, vídeo, áudio ou documento." 
          >
            <template #body>
{
  "to": "5511999999999",
  "mediaType": "image",
  "url": "https://exemplo.com/foto.jpg",
  "caption": "Veja esta foto"
}
            </template>
            <template #response>
{
  "messageId": "3EB0C767D097B7C09D71",
  "timestamp": "2024-01-15T14:31:00Z"
}
            </template>
          </EndpointCard>

          <EndpointCard 
            method="POST" 
            path="/api/instances/:name/send-message-batch" 
            title="Enviar Texto em Lote" 
            description="Envia várias mensagens de texto em paralelo." 
          >
            <template #body>
{
  "messages": [
    { "to": "5511999999999", "text": "Msg 1" },
    { "to": "5511888888888", "text": "Msg 2" }
  ],
  "maxWorkers": 5
}
            </template>
            <template #response>
{
  "total": 2,
  "success": 2,
  "failed": 0,
  "results": [
    { "index": 0, "to": "5511999999999", "messageId": "3EB...", "success": true },
    { "index": 1, "to": "5511888888888", "messageId": "3EB...", "success": true }
  ]
}
            </template>
          </EndpointCard>

          <EndpointCard
            method="POST"
            path="/api/instances/:name/test-webhook"
            title="Testar Webhook"
            description="Envia um evento de teste para a URL de webhook configurada na instância."
          >
            <template #response>
{
  "message": "Test event sent",
  "webhookUrl": "https://api.site.com/webhook"
}
            </template>
          </EndpointCard>

          <EndpointCard
            method="POST"
            path="/api/instances/:name/send-media-batch"
            title="Enviar Mídia em Lote"
            description="Envia várias mídias em paralelo." 
          >
            <template #body>
{
  "messages": [
    { "to": "5511999999999", "mediaType": "image", "url": "https://ex.com/1.jpg" },
    { "to": "5511888888888", "mediaType": "document", "url": "https://ex.com/doc.pdf", "fileName": "doc.pdf" }
  ],
  "maxWorkers": 5
}
            </template>
            <template #response>
{
  "total": 2,
  "success": 2,
  "failed": 0,
  "results": [
    { "index": 0, "to": "5511999999999", "messageId": "3EB...", "success": true },
    { "index": 1, "to": "5511888888888", "messageId": "3EB...", "success": true }
  ]
}
            </template>
          </EndpointCard>
        </section>

        <!-- Webhooks Section -->
        <section v-if="activeTab === 'webhooks'" class="space-y-6">

          <!-- Intro card -->
          <div class="bg-slate-800/50 backdrop-blur-sm border border-slate-700/50 rounded-2xl p-6">
            <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
              <span class="w-8 h-8 rounded-lg bg-emerald-500/10 flex items-center justify-center text-emerald-500">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
              </span>
              Eventos de Webhook
            </h2>
            <p class="text-slate-300 mb-4 leading-relaxed">
              Quando configurado, a API envia requisições <code class="text-emerald-400 bg-emerald-500/10 px-1.5 py-0.5 rounded font-mono text-sm">POST</code> para a sua URL a cada evento. Todos os payloads seguem a mesma estrutura base.
            </p>

            <div class="space-y-4">
              <div>
                <p class="text-slate-400 text-sm font-medium mb-2">Headers enviados em todos os eventos:</p>
                <div class="bg-slate-950 rounded-xl border border-slate-800 p-4 font-mono text-sm overflow-x-auto">
                  <div class="text-slate-400">Content-Type: application/json</div>
                  <div class="text-slate-400">X-Webhook-Event: message.received</div>
                  <div class="text-slate-400">X-Instance-Name: minha-instancia</div>
                </div>
              </div>
              <div>
                <p class="text-slate-400 text-sm font-medium mb-2">Estrutura base de todos os payloads:</p>
                <div class="bg-slate-950 rounded-xl border border-slate-800 p-4 font-mono text-sm overflow-x-auto">
                  <pre class="text-slate-300 whitespace-pre">{
  "event": "message.received",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:30:00Z",
  "data": { ... }
}</pre>
                </div>
              </div>
            </div>
          </div>

          <!-- message.received — texto -->
          <EndpointCard
            method="POST"
            path="message.received → texto"
            title="Mensagem de Texto"
            description="Disparado ao receber uma mensagem de texto simples."
          >
            <template #response>
{
  "event": "message.received",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:30:00Z",
  "data": {
    "from": "5511999999999",
    "fromName": "João Silva",
    "messageId": "3EB0C767D097B7C09D70",
    "messageType": "text",
    "text": "Olá, tudo bem?",
    "isGroup": false,
    "timestamp": 1705335000
  }
}
            </template>
          </EndpointCard>

          <!-- message.received — imagem -->
          <EndpointCard
            method="POST"
            path="message.received → imagem"
            title="Mensagem de Imagem"
            description="Disparado ao receber uma imagem. A mídia é baixada automaticamente e enviada em base64."
          >
            <template #response>
{
  "event": "message.received",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:31:00Z",
  "data": {
    "from": "5511999999999",
    "fromName": "Maria Souza",
    "messageId": "3EB0C767D097B7C09D71",
    "messageType": "image",
    "caption": "Veja essa foto!",
    "mimeType": "image/jpeg",
    "mediaBase64": "iVBORw0KGgoAAAANSUhEUgAA...",
    "isGroup": false,
    "timestamp": 1705335060
  }
}
            </template>
          </EndpointCard>

          <!-- message.received — vídeo -->
          <EndpointCard
            method="POST"
            path="message.received → vídeo"
            title="Mensagem de Vídeo"
            description="Disparado ao receber um vídeo. O conteúdo é entregue em base64."
          >
            <template #response>
{
  "event": "message.received",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:32:00Z",
  "data": {
    "from": "5511999999999",
    "fromName": "Carlos Lima",
    "messageId": "3EB0C767D097B7C09D72",
    "messageType": "video",
    "caption": "Assista este vídeo",
    "mimeType": "video/mp4",
    "mediaBase64": "AAAAIGZ0eXBpc29t...",
    "isGroup": false,
    "timestamp": 1705335120
  }
}
            </template>
          </EndpointCard>

          <!-- message.received — áudio -->
          <EndpointCard
            method="POST"
            path="message.received → áudio"
            title="Mensagem de Áudio"
            description="Disparado ao receber um áudio ou mensagem de voz (PTT). O conteúdo é entregue em base64."
          >
            <template #response>
{
  "event": "message.received",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:33:00Z",
  "data": {
    "from": "5511999999999",
    "fromName": "Ana Costa",
    "messageId": "3EB0C767D097B7C09D73",
    "messageType": "audio",
    "mimeType": "audio/ogg; codecs=opus",
    "mediaBase64": "T2dnUwACAAAAAAAA...",
    "isGroup": false,
    "timestamp": 1705335180
  }
}
            </template>
          </EndpointCard>

          <!-- message.received — documento -->
          <EndpointCard
            method="POST"
            path="message.received → documento"
            title="Mensagem de Documento"
            description="Disparado ao receber um documento (PDF, Word, planilha, etc.). Inclui nome do arquivo."
          >
            <template #response>
{
  "event": "message.received",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:34:00Z",
  "data": {
    "from": "5511999999999",
    "fromName": "Pedro Alves",
    "messageId": "3EB0C767D097B7C09D74",
    "messageType": "document",
    "fileName": "relatorio-mensal.pdf",
    "caption": "Relatório de março",
    "mimeType": "application/pdf",
    "mediaBase64": "JVBERi0xLjQK...",
    "isGroup": false,
    "timestamp": 1705335240
  }
}
            </template>
          </EndpointCard>

          <!-- message.received — sticker -->
          <EndpointCard
            method="POST"
            path="message.received → sticker"
            title="Mensagem de Sticker"
            description="Disparado ao receber um sticker (figurinha). O conteúdo é entregue em base64 no formato WebP."
          >
            <template #response>
{
  "event": "message.received",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:35:00Z",
  "data": {
    "from": "5511999999999",
    "fromName": "Lucia Ferreira",
    "messageId": "3EB0C767D097B7C09D75",
    "messageType": "sticker",
    "mimeType": "image/webp",
    "mediaBase64": "UklGRlYAAABXRUJQ...",
    "isGroup": false,
    "timestamp": 1705335300
  }
}
            </template>
          </EndpointCard>

          <!-- message.received — contato -->
          <EndpointCard
            method="POST"
            path="message.received → contato"
            title="Mensagem de Contato"
            description="Disparado ao receber um cartão de contato. O campo text contém o nome exibido."
          >
            <template #response>
{
  "event": "message.received",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:36:00Z",
  "data": {
    "from": "5511999999999",
    "fromName": "Bruno Neves",
    "messageId": "3EB0C767D097B7C09D76",
    "messageType": "contact",
    "text": "Roberto Santos",
    "isGroup": false,
    "timestamp": 1705335360
  }
}
            </template>
          </EndpointCard>

          <!-- message.received — localização -->
          <EndpointCard
            method="POST"
            path="message.received → localização"
            title="Mensagem de Localização"
            description="Disparado ao receber uma localização compartilhada."
          >
            <template #response>
{
  "event": "message.received",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:37:00Z",
  "data": {
    "from": "5511999999999",
    "fromName": "Fernanda Rocha",
    "messageId": "3EB0C767D097B7C09D77",
    "messageType": "location",
    "isGroup": false,
    "timestamp": 1705335420
  }
}
            </template>
          </EndpointCard>

          <!-- message.received — grupo -->
          <EndpointCard
            method="POST"
            path="message.received → grupo"
            title="Mensagem de Grupo"
            description="Qualquer tipo de mensagem recebida em um grupo. Inclui groupId e groupName adicionalmente."
          >
            <template #response>
{
  "event": "message.received",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:38:00Z",
  "data": {
    "from": "5511999999999",
    "fromName": "Rafael Torres",
    "messageId": "3EB0C767D097B7C09D78",
    "messageType": "text",
    "text": "Bom dia pessoal!",
    "isGroup": true,
    "groupId": "120363123456789012",
    "groupName": "Equipe de Vendas",
    "timestamp": 1705335480
  }
}
            </template>
          </EndpointCard>

          <!-- connection.connected -->
          <EndpointCard
            method="POST"
            path="connection.connected"
            title="Conexão Estabelecida"
            description="Disparado quando a instância se conecta com sucesso ao WhatsApp."
          >
            <template #response>
{
  "event": "connection.connected",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T14:30:00Z",
  "data": {
    "status": "connected",
    "phoneNumber": "5511999999999"
  }
}
            </template>
          </EndpointCard>

          <!-- connection.disconnected -->
          <EndpointCard
            method="POST"
            path="connection.disconnected"
            title="Conexão Encerrada"
            description="Disparado quando a instância perde a conexão com o WhatsApp (queda de rede, reinício, etc.)."
          >
            <template #response>
{
  "event": "connection.disconnected",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T15:00:00Z",
  "data": {
    "status": "disconnected"
  }
}
            </template>
          </EndpointCard>

          <!-- connection.logged_out -->
          <EndpointCard
            method="POST"
            path="connection.logged_out"
            title="Sessão Encerrada (Logout)"
            description="Disparado quando a sessão é encerrada remotamente, por exemplo ao desconectar pelo celular em Dispositivos Vinculados."
          >
            <template #response>
{
  "event": "connection.logged_out",
  "instance": "minha-instancia",
  "timestamp": "2024-01-15T15:10:00Z",
  "data": {
    "status": "logged_out",
    "reason": "Logged out from another device"
  }
}
            </template>
          </EndpointCard>

        </section>
      </div>

      <!-- Quick Links Sidebar -->
      <div class="hidden lg:block space-y-6">
        <div class="bg-slate-800/50 backdrop-blur-sm border border-slate-700/50 rounded-2xl p-6 sticky top-24">
          <h3 class="text-white font-semibold mb-4">Acesso Rápido</h3>
          <ul class="space-y-3 text-sm">
            <li v-for="tab in tabs" :key="tab.id">
              <button 
                @click="activeTab = tab.id"
                :class="[
                  'flex items-center gap-2 transition-colors',
                  activeTab === tab.id ? 'text-emerald-400 font-medium' : 'text-slate-400 hover:text-white'
                ]"
              >
                <span :class="['w-1.5 h-1.5 rounded-full', activeTab === tab.id ? 'bg-emerald-400' : 'bg-slate-600']"></span>
                {{ tab.label }}
              </button>
            </li>
          </ul>

          <div class="mt-8 pt-6 border-t border-slate-700/50">
            <div class="p-4 rounded-xl bg-gradient-to-br from-indigo-500/10 to-purple-500/10 border border-indigo-500/20">
              <h4 class="text-indigo-400 font-medium mb-1">Precisa de ajuda?</h4>
              <p class="text-xs text-indigo-300/80">Entre em contato com o suporte técnico para dúvidas avançadas.</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const activeTab = ref('auth')

const tabs = [
  { id: 'auth', label: 'Autenticação' },
  { id: 'instances', label: 'Instâncias' },
  { id: 'connection', label: 'Conexão' },
  { id: 'messages', label: 'Mensagens' },
  { id: 'webhooks', label: 'Webhooks' },
]
</script>

<style>
.animate-fade-in {
  animation: fadeIn 0.5s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>

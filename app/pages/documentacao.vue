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
          />
          
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
  "proxyEnabled": true,
  "proxyUrl": "http://proxy.com:8080"
}
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
          />
          
          <EndpointCard 
            method="GET" 
            path="/api/instances/:name/wa-status" 
            title="Status da Conexão" 
            description="Verifica se o WhatsApp está conectado." 
          />
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
  "simulateTyping": true
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
  { id: 'tags', label: 'Etiquetas' },
  { id: 'webhooks', label: 'Webhooks' }
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

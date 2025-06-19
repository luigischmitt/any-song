# AnySong - Setup e Organização

## 🎯 O que foi organizado

### ✅ Funcionalidades implementadas:
1. **Login/Autenticação** - Sistema simples de auth com Firebase
2. **YouTube Downloader** - Download de músicas do YouTube (`utils/yt_downloader.py`)
3. **Upload de música** - Frontend pronto para upload
4. **Processamento completo** - Novo sistema integrado (`process_song.py`)

### 🗂️ Estrutura de pastas organizada:

Quando você faz upload de uma música, o sistema cria uma estrutura assim:
```
output/
├── Artista - Música/
│   ├── original_arquivo.mp3          # Arquivo original
│   ├── backing_track_info.json       # Info do backing track do Music.AI
│   ├── Artista - Música_lyrics.json  # Letras completas
│   ├── Artista - Música_lyrics.txt   # Letras em texto
│   └── processing_summary.json       # Resumo do processamento
```

## 🔧 Arquivos importantes:

### Backend Python:
- `process_song.py` - **NOVO**: Processamento principal integrado
- `utils/backing_track.py` - Integração com Music.AI
- `utils/catch_lyrics.py` - Busca letras no MusixMatch
- `utils/yt_downloader.py` - Download do YouTube
- `utils/song_name.py` - Extração de info dos arquivos

### Backend Go:
- `cmd/api/main.go` - API principal com auth e endpoints
- `Dockerfile` - Build simplificado

### Frontend:
- Estrutura React/Next.js já pronta
- Upload funcionando
- Auth integrado

## 🚀 Como usar:

### 1. Configurar variáveis de ambiente:
```bash
# Criar .env no backend/
MUSICAI_API_KEY=sua_chave_aqui
MUSIXMATCH_API_KEY=sua_chave_aqui
```

### 2. Rodar com Docker:
```bash
# Backend
cd backend
sudo docker build -t any-song-backend .
sudo docker run -p 8000:8000 any-song-backend

# Frontend
cd frontend
npm install
npm run dev
```

### 3. Fluxo de uso:
1. **Login** - Usuário faz login
2. **Upload** - Envia arquivo de música
3. **Processamento automático**:
   - Salva arquivo original
   - Gera backing track via Music.AI
   - Busca letras no MusixMatch
   - Organiza tudo em pastas
4. **Resultado** - Pasta com todos os arquivos organizados

## 📁 O que foi removido:

### ❌ Arquivos desnecessários removidos:
- `lsync/` - Pasta inteira de sincronização complexa
- `karaokefy.py` - Processamento complexo desnecessário
- `cochichando.py` - Transcrição não necessária agora
- `audio_synch.py` - Sincronização não usada
- Dependências pesadas (torch, demucs, etc.)

### ✅ Dependências simplificadas:
```
yt-dlp>=2023.12.30
pydub>=0.25.1
mutagen>=1.47.0
requests>=2.28.2
python-dotenv>=1.0.0
ffmpeg-python>=0.2.0
musicai-sdk>=1.0.0
musicxmatch-api>=1.0.0
shazamio>=0.4.0
```

## 🎵 Exemplo de uso:

```bash
# Testar processamento direto:
cd backend
python3 process_song.py "caminho/para/musica.mp3" "Título" "Artista"

# Resultado JSON:
{
  "success": true,
  "song_title": "Título",
  "artist_name": "Artista",
  "song_folder": "output/Artista - Título",
  "backing_track_url": "https://musicai.com/result/...",
  "lyrics": "Letra da música...",
  "files": {
    "backing_track_info": "...",
    "lyrics_json": "...",
    "lyrics_txt": "...",
    "summary": "..."
  }
}
```

## 🔄 Próximos passos:

1. **Configurar APIs** - Music.AI e MusixMatch
2. **Testar upload** - Frontend → Backend
3. **Implementar bucket** - AWS S3 ou similar
4. **Adicionar sincronização** - Quando necessário

## 🐛 Troubleshooting:

### Docker build falha:
- Verificar se todas as dependências estão no requirements.txt
- Usar `sudo` se necessário

### APIs não funcionam:
- Verificar chaves de API no .env
- Testar conectividade

### Upload não funciona:
- Verificar CORS no backend
- Verificar endpoints no frontend 
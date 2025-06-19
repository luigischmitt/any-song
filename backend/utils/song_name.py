import asyncio
import re
import os
from shazamio import Shazam

def extract_song_info(filename: str) -> dict:
    """
    Extrai informações de título e artista do nome do arquivo
    
    Args:
        filename (str): Nome do arquivo
    
    Returns:
        dict: Dicionário com 'title' e 'artist' extraídos
    """
    # Remove extensão
    name = os.path.splitext(filename)[0]
    
    # Remove UUID se presente (padrão: 8-4-4-4-12 caracteres hexadecimais)
    uuid_pattern = r'^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}_'
    name = re.sub(uuid_pattern, '', name, flags=re.IGNORECASE)
    
    # Padrões comuns para extrair artista e título
    patterns = [
        r'^(.+?)\s*-\s*(.+)$',  # Artista - Título
        r'^(.+?)\s*–\s*(.+)$',  # Artista – Título (traço longo)
        r'^(.+?)\s*:\s*(.+)$',  # Artista: Título
        r'^(.+?)\s+by\s+(.+)$', # Título by Artista (invertido)
        r'^(.+?)\s*\|\s*(.+)$', # Artista | Título
    ]
    
    for pattern in patterns:
        match = re.search(pattern, name, re.IGNORECASE)
        if match:
            if 'by' in pattern:  # Caso "Título by Artista"
                return {
                    'title': match.group(1).strip(),
                    'artist': match.group(2).strip()
                }
            else:  # Caso "Artista - Título"
                return {
                    'title': match.group(2).strip(),
                    'artist': match.group(1).strip()
                }
    
    # Se não encontrou padrão, tentar detectar se tem palavras como "Official", "Music Video", etc.
    # que geralmente indicam que é um título de vídeo do YouTube
    video_indicators = ['official', 'music video', 'mv', 'video', 'clip', 'lyrics']
    name_lower = name.lower()
    
    if any(indicator in name_lower for indicator in video_indicators):
        # Tentar extrair artista do início
        words = name.split()
        if len(words) >= 2:
            # Assumir que as primeiras 1-2 palavras são o artista
            if len(words[0]) > 2:  # Se a primeira palavra é longa o suficiente
                return {
                    'title': name,
                    'artist': words[0].strip()
                }
            elif len(words) >= 2:  # Usar as duas primeiras palavras como artista
                return {
                    'title': name,
                    'artist': f"{words[0]} {words[1]}".strip()
                }
    
    # Fallback: usar o nome completo como título e "Artista Desconhecido"
    return {
        'title': name.strip(),
        'artist': 'Artista Desconhecido'
    }

async def get_song_name(path_original_song: str):
    shazam = Shazam()
    out = await shazam.recognize(path_original_song)  
    print(out)
    if out['matches'] != []:
        return out['track']['title'], out['track']['subtitle']
    else:
        return None, None


if __name__ == "__main__":
    print(asyncio.run(get_song_name("/root/any-song/backend/utils/audios/songs/ROSÉ & Bruno Mars - APT. (Official Music Video).mp3")))
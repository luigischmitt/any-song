import { useState, useCallback } from 'react';
import styles from './Upload.module.css';

export default function Upload({ onFileUpload }) {
  const [dragActive, setDragActive] = useState(false);
  const [uploadedFile, setUploadedFile] = useState(null);
  const [processing, setProcessing] = useState(false);
  const [processingStatus, setProcessingStatus] = useState('');
  const [jobId, setJobId] = useState(null);
  const [result, setResult] = useState(null);

  const handleDrag = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      const file = e.dataTransfer.files[0];
      handleFile(file);
    }
  }, []);

  const handleFileInput = (e) => {
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0];
      handleFile(file);
    }
  };

  const handleFile = (file) => {
    // Verificar se é um arquivo de áudio válido
    const validTypes = ['audio/mpeg', 'audio/wav', 'audio/mp3'];
    const validExtensions = ['.mp3', '.wav'];
    
    const isValidType = validTypes.includes(file.type) || 
                       validExtensions.some(ext => file.name.toLowerCase().endsWith(ext));

    if (!isValidType) {
      alert('Please upload only .mp3 or .wav files');
      return;
    }

    setUploadedFile(file);
    setResult(null);
    setJobId(null);
    if (onFileUpload) {
      onFileUpload(file);
    }
  };

  const removeFile = () => {
    setUploadedFile(null);
    setResult(null);
    setJobId(null);
    setProcessing(false);
    setProcessingStatus('');
    if (onFileUpload) {
      onFileUpload(null);
    }
  };

  const convertToKaraoke = async () => {
    if (!uploadedFile) return;

    setProcessing(true);
    setProcessingStatus('Uploading file...');

    try {
      const formData = new FormData();
      formData.append('audio_file', uploadedFile);
      formData.append('song_title', uploadedFile.name.replace(/\.[^/.]+$/, ""));
      formData.append('artist_name', '');

      const token = localStorage.getItem('token');
      const response = await fetch('http://localhost:8000/karaoke/process', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
        body: formData,
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      
      if (data.success) {
        setJobId(data.job_id);
        setProcessingStatus('Processing with AI...');
        
        // Polling para verificar status
        pollJobStatus(data.job_id);
      } else {
        throw new Error(data.message || 'Processing failed');
      }
    } catch (error) {
      console.error('Error processing karaoke:', error);
      setProcessingStatus('Error: ' + error.message);
      setProcessing(false);
    }
  };

  const pollJobStatus = async (jobId) => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`http://localhost:8000/karaoke/status/${jobId}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      
      if (data.status === 'completed') {
        setProcessing(false);
        setProcessingStatus('✅ Karaoke ready!');
        setResult({
          backingTrackUrl: data.backing_track_url,
          originalUrl: data.original_url
        });
      } else if (data.status === 'failed') {
        setProcessing(false);
        setProcessingStatus('❌ Processing failed: ' + (data.error_message || 'Unknown error'));
      } else {
        // Continue polling
        setProcessingStatus('🎵 Creating backing track...');
        setTimeout(() => pollJobStatus(jobId), 3000);
      }
    } catch (error) {
      console.error('Error checking job status:', error);
      setProcessingStatus('Error checking status: ' + error.message);
      setProcessing(false);
    }
  };

  return (
    <div className={styles.uploadContainer}>
      <h3 className={styles.title}>Upload Your Audio File</h3>
      
      {!uploadedFile ? (
        <div
          className={`${styles.dropZone} ${dragActive ? styles.active : ''}`}
          onDragEnter={handleDrag}
          onDragLeave={handleDrag}
          onDragOver={handleDrag}
          onDrop={handleDrop}
        >
          <div className={styles.dropContent}>
            <div className={styles.icon}>🎵</div>
            <p className={styles.dropText}>
              Drag & drop your audio file here
            </p>
            <p className={styles.supportedFormats}>
              Supported formats: MP3, WAV
            </p>
            <div className={styles.orDivider}>
              <span>or</span>
            </div>
            <label htmlFor="file-input" className={styles.browseButton}>
              Browse Files
            </label>
            <input
              id="file-input"
              type="file"
              accept=".mp3,.wav,audio/*"
              onChange={handleFileInput}
              className={styles.fileInput}
            />
          </div>
        </div>
      ) : (
        <div className={styles.filePreview}>
          <div className={styles.fileInfo}>
            <div className={styles.fileIcon}>🎵</div>
            <div className={styles.fileDetails}>
              <p className={styles.fileName}>{uploadedFile.name}</p>
              <p className={styles.fileSize}>
                {(uploadedFile.size / 1024 / 1024).toFixed(2)} MB
              </p>
            </div>
            <button 
              onClick={removeFile}
              className={styles.removeButton}
              title="Remove file"
              disabled={processing}
            >
              ✕
            </button>
          </div>
          
          {!processing && !result && (
            <div className={styles.processButton}>
              <button 
                className={styles.convertButton}
                onClick={convertToKaraoke}
                disabled={processing}
              >
                Convert to Karaoke
              </button>
            </div>
          )}

          {processing && (
            <div className={styles.processingStatus}>
              <div className={styles.spinner}></div>
              <p>{processingStatus}</p>
            </div>
          )}

          {result && (
            <div className={styles.result}>
              <h4>✅ Karaoke Ready!</h4>
              <div className={styles.resultButtons}>
                <a 
                  href={result.backingTrackUrl} 
                  download
                  className={styles.downloadButton}
                >
                  Download Backing Track
                </a>
                <button 
                  className={styles.playButton}
                  onClick={() => window.open(`/karaoke?track=${encodeURIComponent(result.backingTrackUrl)}`, '_blank')}
                >
                  Play Karaoke
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
} 
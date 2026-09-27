import { useTranslation } from 'react-i18next'
import { USBIcon, RAMIcon } from 'icons'
import { FormControlLabel, MenuItem, Switch } from '@material-ui/core'
import TextField from '@material-ui/core/TextField'

import {
  CacheLegendGrid,
  CacheLegendDot,
  MainSettingsContent,
  StorageButton,
  StorageIconWrapper,
  CacheStorageSelector,
  SettingSectionLabel,
  PreloadCachePercentage,
  cacheBeforeReaderColor,
  cacheAfterReaderColor,
} from './style'
import SliderInput from './SliderInput'

const CacheStorageLocationLabel = ({ style }) => {
  const { t } = useTranslation()

  return (
    <SettingSectionLabel style={style}>
      {t('SettingsDialog.CacheStorageLocation')}
      <small>{t('SettingsDialog.UseDiskDesc')}</small>
    </SettingSectionLabel>
  )
}

export default function PrimarySettingsComponent({
  settings,
  inputForm,
  cachePercentage,
  preloadCachePercentage,
  cacheSize,
  isProMode,
  setCacheSize,
  setCachePercentage,
  setPreloadCachePercentage,
  updateSettings,
}) {
  const { t } = useTranslation()
  const { UseDisk, TorrentsSavePath, RemoveCacheOnDrop, PadTailPartial } = settings || {}
  const preloadCacheSize = Math.round((cacheSize / 100) * preloadCachePercentage)

  return (
    <MainSettingsContent>
      <div>
        <SettingSectionLabel>{t('SettingsDialog.CacheSettings')}</SettingSectionLabel>

        <PreloadCachePercentage
          value={100 - cachePercentage}
          label={`${t('Cache')} ${cacheSize} ${t('MB')}`}
          preloadCachePercentage={preloadCachePercentage}
        />

        <CacheLegendGrid>
          <CacheLegendDot color={cacheBeforeReaderColor} aria-hidden />
          <div className='cache-legend-value'>
            {100 - cachePercentage}% ({Math.round((cacheSize / 100) * (100 - cachePercentage))} {t('MB')})
          </div>
          <div className='cache-legend-desc'>{t('SettingsDialog.CacheBeforeReaderDesc')}</div>

          <CacheLegendDot color={cacheAfterReaderColor} aria-hidden />
          <div className='cache-legend-value'>
            {cachePercentage}% ({Math.round((cacheSize / 100) * cachePercentage)} {t('MB')})
          </div>
          <div className='cache-legend-desc'>{t('SettingsDialog.CacheAfterReaderDesc')}</div>
        </CacheLegendGrid>

        <br />

        <TextField
          select
          fullWidth
          variant='outlined'
          label='Flow cache preset'
          value={[256, 512, 1024, 2048, 4096].includes(Number(cacheSize)) ? Number(cacheSize) : 'custom'}
          onChange={event => {
            if (event.target.value !== 'custom') setCacheSize(Number(event.target.value))
          }}
          helperText='Choose a RAM budget based on the media and available memory, or enter a custom size below.'
        >
          <MenuItem value={256}>256 MB · Balanced</MenuItem>
          <MenuItem value={512}>512 MB · High</MenuItem>
          <MenuItem value={1024}>1024 MB · REMUX</MenuItem>
          <MenuItem value={2048}>2048 MB · REMUX+</MenuItem>
          <MenuItem value={4096}>4096 MB · Extreme</MenuItem>
          <MenuItem value='custom'>Custom</MenuItem>
        </TextField>

        <br />

        <SliderInput
          isProMode
          title={t('SettingsDialog.CacheSize')}
          value={cacheSize}
          setValue={setCacheSize}
          sliderMin={32}
          sliderMax={4096}
          inputMin={32}
          inputMax={999999}
          step={4}
          onBlurCallback={value => setCacheSize(Math.round(value / 4) * 4)}
        />

        <SliderInput
          isProMode={isProMode}
          title={t('SettingsDialog.ReaderReadAHead')}
          value={cachePercentage}
          setValue={setCachePercentage}
          sliderMin={40}
          sliderMax={95}
          inputMin={0}
          inputMax={100}
        />

        <SliderInput
          isProMode={isProMode}
          title={`${t('SettingsDialog.PreloadCache')} - ${preloadCachePercentage}% (${preloadCacheSize} ${t('MB')})`}
          value={preloadCachePercentage}
          setValue={setPreloadCachePercentage}
          sliderMin={0}
          sliderMax={100}
          inputMin={0}
          inputMax={100}
        />

        <FormControlLabel
          control={<Switch checked={!!PadTailPartial} onChange={inputForm} id='PadTailPartial' color='secondary' />}
          label={t('SettingsDialog.PadTailPartial')}
        />
        <small style={{ display: 'block', opacity: 0.7 }}>{t('SettingsDialog.PadTailPartialHint')}</small>
      </div>

      {UseDisk ? (
        <div>
          <CacheStorageLocationLabel />

          <div style={{ display: 'grid', gridAutoFlow: 'column' }}>
            <StorageButton small onClick={() => updateSettings({ UseDisk: false })}>
              <StorageIconWrapper small>
                <RAMIcon color='#323637' />
              </StorageIconWrapper>

              <div>{t('SettingsDialog.RAM')}</div>
            </StorageButton>

            <StorageButton small selected>
              <StorageIconWrapper small selected>
                <USBIcon color='#dee3e5' />
              </StorageIconWrapper>

              <div>{t('SettingsDialog.Disk')}</div>
            </StorageButton>
          </div>

          <FormControlLabel
            control={
              <Switch checked={RemoveCacheOnDrop} onChange={inputForm} id='RemoveCacheOnDrop' color='secondary' />
            }
            label={t('SettingsDialog.RemoveCacheOnDrop')}
            labelPlacement='start'
          />
          <div>
            <small>{t('SettingsDialog.RemoveCacheOnDropDesc')}</small>
          </div>
          <br />
          <TextField
            onChange={inputForm}
            margin='normal'
            id='TorrentsSavePath'
            label={t('SettingsDialog.TorrentsSavePath')}
            value={TorrentsSavePath}
            type='url'
            variant='outlined'
            fullWidth
          />
        </div>
      ) : (
        <CacheStorageSelector>
          <CacheStorageLocationLabel style={{ placeSelf: 'start', gridArea: 'label' }} />

          <StorageButton selected>
            <StorageIconWrapper selected>
              <RAMIcon color='#dee3e5' />
            </StorageIconWrapper>

            <div>{t('SettingsDialog.RAM')}</div>
          </StorageButton>

          <StorageButton onClick={() => updateSettings({ UseDisk: true })}>
            <StorageIconWrapper>
              <USBIcon color='#323637' />
            </StorageIconWrapper>

            <div>{t('SettingsDialog.Disk')}</div>
          </StorageButton>
        </CacheStorageSelector>
      )}
    </MainSettingsContent>
  )
}

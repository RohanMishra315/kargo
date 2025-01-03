import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faQuestionCircle } from '@fortawesome/free-solid-svg-icons';

import { SegmentLabel } from '@ui/features/common/segment-label';
import { DESCRIPTION_ANNOTATION_KEY } from '@ui/features/common/utils';
import { Secret } from '@ui/gen/k8s.io/api/core/v1/generated_pb';

import { CredentialTypeLabelKey, CredentialsDataKey, CredentialsType } from './types';

export const typeLabel = (type: CredentialsType) => (
  <SegmentLabel icon={iconForCredentialsType(type)}>{type.toUpperCase()}</SegmentLabel>
);

export const iconForCredentialsType = (type: CredentialsType) => {
  switch (type) {
    case 'git':
      return faGit;
    case 'helm':
      return faAnchor;
    case 'image':
      return faDocker;
    default:
      return faQuestionCircle;
  }
};

export const labelForKey = (s: string) =>
  s
    .split('')
    .map((c, i) => (c === c.toUpperCase() ? (i !== 0 ? ' ' : '') + c : c))
    .join('')
    .replace(/^./, (str) => str.toUpperCase())
    .replace('Url', 'URL');

export const constructDefaults = (init?: Secret) => {
  if (!init) {
    return {
      name: '',
      description: '',
      type: 'git',
      repoUrl: '',
      repoUrlIsRegex: false,
      username: '',
      password: ''
    };
  }
  return {
    name: init?.metadata?.name || '',
    description: init?.metadata?.annotations[DESCRIPTION_ANNOTATION_KEY],
    type: init?.metadata?.labels[CredentialTypeLabelKey] || 'git',
    repoUrl: init?.stringData[CredentialsDataKey.RepoUrl],
    repoUrlIsRegex: init?.stringData[CredentialsDataKey.RepoUrlIsRegex] === 'true',
    username: init?.stringData[CredentialsDataKey.Username],
    password: ''
  };
};

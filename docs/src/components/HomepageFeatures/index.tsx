import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  emoji: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Modular Agentic Runtimes',
    emoji: '🤖',
    description: (
      <>
        Switch between the artisan Vibe Agent and the powerhouse Copilot SDK 
        native engine on the fly.
      </>
    ),
  },
  {
    title: 'System-Intimate Tooling',
    emoji: '🛠️',
    description: (
      <>
        Deep access to your filesystem, system resources, and Git state for 
        autonomous engineering that actually works.
      </>
    ),
  },
  {
    title: 'Directory-Aware Sessions',
    emoji: '📂',
    description: (
      <>
        Project isolation is built-in. Sessions are keyed to your directory hash, 
        ensuring your projects stay clean and separated.
      </>
    ),
  },
];

function Feature({title, emoji, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <span style={{ fontSize: '4rem' }}>{emoji}</span>
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}

// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
	site: 'https://vardius.github.io',
	base: '/progress-go',
	trailingSlash: 'always',
	// Pages import the runnable programs in ../examples, so the dev server may read outside this folder.
	vite: { server: { fs: { allow: ['..'] } } },
	integrations: [
		starlight({
			title: 'progress-go',
			description:
				'A progress bar for Go: redrawn in place on a terminal, safe to advance from many goroutines, and free of allocations while it runs.',
			logo: { src: './src/assets/mark.svg', alt: '' },
			favicon: '/favicon.svg',
			components: { Hero: './src/components/Hero.astro' },
			// Long lines wrap instead of scrolling under the copy button on phones.
			expressiveCode: { defaultProps: { wrap: true } },
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/vardius/progress-go' }],
			customCss: ['./src/styles/custom.css'],
			sidebar: [
				{
					label: 'Start here',
					items: ['getting-started'],
				},
				{
					label: 'Guides',
					items: ['counting', 'output', 'goroutines'],
				},
				{
					label: 'Reference',
					items: [
						'benchmarks',
						'migrating-from-the-untagged-version',
						{ label: 'API reference', link: 'https://pkg.go.dev/github.com/vardius/progress-go' },
					],
				},
			],
		}),
	],
});

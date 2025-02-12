<script>
	import { injectAnalytics } from '@vercel/analytics/sveltekit';
	import { onMount } from 'svelte';
	import Hero from './Hero.svelte';
	import Projects from './Projects.svelte';
	import Blog from './Blog.svelte';
	import Newsletter from './Newsletter.svelte';

	let isScrolled = false;
	let activeSection = 'hero';

	onMount(() => {
		injectAnalytics();
		window.addEventListener('scroll', () => {
			isScrolled = window.scrollY > 20;

			// Determine the active section based on scroll position
			const sections = document.querySelectorAll('section');
			sections.forEach((section) => {
				const rect = section.getBoundingClientRect();
				if (rect.top <= 100 && rect.bottom >= 100) {
					activeSection = section.id;
				}
			});
		});
	});

	// Smooth scroll function
	function scrollToSection(sectionId) {
		const element = document.getElementById(sectionId);
		element?.scrollIntoView({ behavior: 'smooth' });
	}
</script>

<!-- Fixed Navbar -->
<nav
	class="fixed right-6 top-6 z-50 flex flex-wrap items-center justify-center gap-4 rounded-full px-6 py-3 text-sm transition-all duration-300 md:right-12 md:top-8 md:gap-8 md:px-8 md:py-4"
	class:bg-white={isScrolled}
	class:shadow-lg={isScrolled}
>
	<ul class="flex flex-wrap items-center justify-center gap-4 md:gap-8">
		{#each ['about', 'projects', 'blog', 'newsletter'] as section}
			<li>
				<button
					on:click={() => scrollToSection(section)}
					class="alexandria relative text-gray-500 transition-colors duration-300 hover:text-black md:text-lg"
					class:text-black={activeSection === section}
				>
					{section.charAt(0).toUpperCase() + section.slice(1)}
					{#if activeSection === section}
						<div
							class="absolute -bottom-1 left-0 h-0.5 w-full bg-black transition-all duration-300"
						></div>
					{/if}
				</button>
			</li>
		{/each}
	</ul>
</nav>

<main class="bg-white">
	<!-- Hero Section -->
	<section id="about" class="flex min-h-screen flex-col items-center justify-center text-center">
		<Hero />
	</section>

	<!-- Projects Section -->
	<section id="projects" class="flex min-h-screen flex-col items-center justify-center text-center">
		<Projects />
	</section>

	<!-- Blog Section -->
	<section id="blog" class="flex min-h-screen flex-col items-center justify-center text-center">
		<Blog />
	</section>

	<!-- Newsletter Section -->
	<section
		id="newsletter"
		class="flex min-h-screen flex-col items-center justify-center text-center"
	>
		<Newsletter />
	</section>
</main>

<style>
	:global(body) {
		background-color: #faf8f6;
		scroll-behavior: smooth;
		margin: 0;
		padding: 0;
	}

	:global(.bg-coral-400) {
		background-color: #ff906d;
	}

	:global(.bg-coral-500) {
		background-color: #ff7f5d;
	}

	section {
		scroll-margin-top: 4rem;
	}

	nav {
		backdrop-filter: blur(8px);
		background-color: rgba(255, 255, 255, 0.8);
	}

	nav:not(.bg-white) {
		background-color: transparent;
	}

	@media (max-width: 768px) {
		nav {
			right: 50%;
			transform: translateX(50%);
			padding: 0.5rem 1rem;
		}

		ul {
			gap: 6px;
		}

		section {
			padding: 2rem 1rem;
		}
	}
</style>

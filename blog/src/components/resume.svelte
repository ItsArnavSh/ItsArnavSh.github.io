<script lang="ts">
	import '../routes/layout.css';
</script>

<div class="relative flex items-center justify-center h-[300px] overflow-visible">
	<!-- Background text -->
	<p class="text-(--lightbg) text-lg lux-roman text-center leading-relaxed">
		A résumé or resume (or alternatively <br />resumé) is a document created and used <br />by a
		person to present their background<br />, skills, and accomplishments. Résumés are<br /> most
		often used to secure a job, whether in<br /> the same organization or another.
	</p>

	<!-- Overlay pill button -->
	<a
		href="/projects"
		class="absolute z-10 bg-(--accent) text-black text-4xl share-tech
		       px-20 py-6 rounded-[999px] drop-shadow-[8px_6px_0px_rgba(0,0,0,1)]
		       transition-all duration-300 hover:scale-100 hover:drop-shadow-[20px_16px_0px_rgba(0,0,0,1)]"
	>
		Résumé?
	</a>
</div>
